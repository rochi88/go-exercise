package utils

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"go-boilerplate/internal/shared/cache"
)

// DeviceInfo represents device information needed for session tracking
type DeviceInfo struct {
	// Basic device info
	UserAgent string `json:"user_agent"`
	IPAddress string `json:"ip_address"`

	// Parsed device info
	OS         string `json:"os"` // windows, macos, linux, ios, android
	OSVersion  string `json:"os_version"`
	Browser    string `json:"browser"` // chrome, firefox, safari, edge
	BrowserVer string `json:"browser_version"`

	// Formatted device name: "Browser (OS) - OS_Version"
	DeviceName string `json:"device_name"`

	// Location info
	Country  string `json:"country"`
	Region   string `json:"region"`
	City     string `json:"city"`
	Timezone string `json:"timezone"`
	ISP      string `json:"isp"`

	// Security info
	IsBot     bool `json:"is_bot"`
	RiskScore int  `json:"risk_score"` // 0-100, higher = more suspicious

	// Session info
	Fingerprint string `json:"fingerprint"`
}

// DetectDevice extracts device information from HTTP request
func DetectDevice(r *http.Request) *DeviceInfo {
	userAgent := r.Header.Get("User-Agent")
	ipAddress := GetClientIP(r)

	deviceInfo := &DeviceInfo{
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}

	// Parse user agent
	parseUserAgent(deviceInfo)

	// Detect location from IP
	detectLocation(deviceInfo)

	// Generate formatted device name
	deviceInfo.DeviceName = formatDeviceName(deviceInfo)

	// Detect security risks
	detectSecurityRisks(deviceInfo)

	// Generate device fingerprint
	deviceInfo.Fingerprint = generateFingerprint(deviceInfo)

	return deviceInfo
}

// formatDeviceName creates device name in format: "Browser (OS) - OS_Version"
func formatDeviceName(info *DeviceInfo) string {
	browser := capitalizeFirst(info.Browser)
	if browser == "Unknown" {
		browser = "Unknown Browser"
	}

	os := capitalizeFirst(info.OS)
	if os == "Unknown" {
		os = "Unknown OS"
	}

	deviceName := fmt.Sprintf("%s (%s)", browser, os)

	if info.OSVersion != "" {
		deviceName += " - " + info.OSVersion
	}

	return deviceName
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return "Unknown"
	}
	if s == "macos" {
		return "macOS"
	}
	if s == "ios" {
		return "iOS"
	}
	// Simple capitalization without using deprecated strings.Title
	if len(s) == 1 {
		return strings.ToUpper(s)
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// GetClientIP extracts the real client IP from various headers
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (most common)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP (original client)
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" && net.ParseIP(xri) != nil {
		return xri
	}

	// Check CF-Connecting-IP (Cloudflare)
	cfip := r.Header.Get("CF-Connecting-IP")
	if cfip != "" && net.ParseIP(cfip) != nil {
		return cfip
	}

	// Check X-Client-IP
	xcip := r.Header.Get("X-Client-IP")
	if xcip != "" && net.ParseIP(xcip) != nil {
		return xcip
	}

	// Fallback to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// parseUserAgent parses the User-Agent string to extract device information
func parseUserAgent(info *DeviceInfo) {
	ua := strings.ToLower(info.UserAgent)

	// Detect bots first
	if isBot(ua) {
		info.IsBot = true
		info.Browser = "bot"
		info.OS = "unknown"
		return
	}

	// Detect OS
	info.OS = detectOS(ua)
	info.OSVersion = detectOSVersion(ua, info.OS)

	// Detect browser
	info.Browser = detectBrowser(ua)
	info.BrowserVer = detectBrowserVersion(ua, info.Browser)
}

// isBot checks if the user agent is from a bot
func isBot(ua string) bool {
	botPatterns := []string{
		"bot", "crawler", "spider", "scraper", "indexer",
		"googlebot", "bingbot", "yahoo", "duckduckbot",
		"facebookexternalhit", "twitterbot", "linkedinbot",
		"whatsapp", "telegrambot", "discordbot",
	}

	for _, pattern := range botPatterns {
		if strings.Contains(ua, pattern) {
			return true
		}
	}
	return false
}

// detectOS detects the operating system from user agent
func detectOS(ua string) string {
	osPatterns := map[string][]string{
		"windows":  {"windows nt", "win32", "win64"},
		"macos":    {"macintosh", "mac os x", "macos"},
		"linux":    {"linux", "ubuntu", "fedora", "centos", "debian"},
		"ios":      {"iphone", "ipad", "ipod"},
		"android":  {"android"},
		"chromeos": {"cros"},
	}

	for os, patterns := range osPatterns {
		for _, pattern := range patterns {
			if strings.Contains(ua, pattern) {
				return os
			}
		}
	}
	return "unknown"
}

// detectOSVersion extracts OS version from user agent
func detectOSVersion(ua, os string) string {
	switch os {
	case "windows":
		if match := regexp.MustCompile(`windows nt (\d+\.\d+)`).FindStringSubmatch(ua); len(match) > 1 {
			return match[1]
		}
	case "macos":
		if match := regexp.MustCompile(`mac os x (\d+[_\d]+)`).FindStringSubmatch(ua); len(match) > 1 {
			return strings.ReplaceAll(match[1], "_", ".")
		}
	case "ios":
		if match := regexp.MustCompile(`os (\d+[_\d]+)`).FindStringSubmatch(ua); len(match) > 1 {
			return strings.ReplaceAll(match[1], "_", ".")
		}
	case "android":
		if match := regexp.MustCompile(`android (\d+\.\d+)`).FindStringSubmatch(ua); len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

// detectBrowser detects the browser from user agent
func detectBrowser(ua string) string {
	browserPatterns := map[string][]string{
		"chrome":  {"chrome", "chromium"},
		"firefox": {"firefox", "fxios"},
		"safari":  {"safari", "webkit"},
		"edge":    {"edg", "edge"},
		"opera":   {"opera", "opr"},
		"ie":      {"msie", "trident"},
	}

	for browser, patterns := range browserPatterns {
		for _, pattern := range patterns {
			if strings.Contains(ua, pattern) {
				return browser
			}
		}
	}
	return "unknown"
}

// detectBrowserVersion extracts browser version from user agent
func detectBrowserVersion(ua, browser string) string {
	switch browser {
	case "chrome":
		if match := regexp.MustCompile(`chrome/(\d+\.\d+)`).FindStringSubmatch(ua); len(match) > 1 {
			return match[1]
		}
	case "firefox":
		if match := regexp.MustCompile(`firefox/(\d+\.\d+)`).FindStringSubmatch(ua); len(match) > 1 {
			return match[1]
		}
	case "safari":
		if match := regexp.MustCompile(`version/(\d+\.\d+)`).FindStringSubmatch(ua); len(match) > 1 {
			return match[1]
		}
	case "edge":
		if match := regexp.MustCompile(`edg/(\d+\.\d+)`).FindStringSubmatch(ua); len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

// detectDeviceModel extracts device model from user agent
// detectSecurityRisks analyzes the device info for security risks
func detectSecurityRisks(info *DeviceInfo) {
	riskScore := 0

	// Check for VPN patterns in IP
	if isVPNIP(info.IPAddress) {
		riskScore += 20
	}

	// Check for proxy headers
	if hasProxyHeaders(info) {
		riskScore += 15
	}

	// Check for Tor exit nodes
	if isTorIP(info.IPAddress) {
		riskScore += 30
	}

	// Check for suspicious user agents
	if isSuspiciousUA(info.UserAgent) {
		riskScore += 25
	}

	// Check for bot behavior
	if info.IsBot {
		riskScore += 10
	}

	// Cap risk score at 100
	if riskScore > 100 {
		riskScore = 100
	}

	info.RiskScore = riskScore
}

// isVPNIP checks if IP is likely from a VPN
func isVPNIP(ip string) bool {
	// This is a simplified check - in production, you'd use a VPN detection service
	vpnPatterns := []string{
		"10.", "172.16.", "192.168.", // Private networks
	}

	for _, pattern := range vpnPatterns {
		if strings.HasPrefix(ip, pattern) {
			return true
		}
	}
	return false
}

// hasProxyHeaders checks for proxy-related headers
func hasProxyHeaders(info *DeviceInfo) bool {
	// This would check for various proxy headers in the original request
	// For now, we'll use a simple heuristic
	return strings.Contains(strings.ToLower(info.UserAgent), "proxy")
}

// isTorIP checks if IP is a Tor exit node
func isTorIP(ip string) bool {
	// In production, you would maintain a cached list of Tor exit nodes
	// This could be updated periodically from https://check.torproject.org/exit-addresses
	// For now, we'll implement a basic check using known Tor exit node patterns

	// Parse IP address
	if net.ParseIP(ip) == nil {
		return false
	}

	// Check against known Tor exit node ranges (simplified example)
	// In production, you'd load this from a database or API
	torRanges := []string{
		"185.220.100.0/24", // Example Tor exit node range
		"185.220.101.0/24",
		"185.220.102.0/24",
		"199.249.223.0/24",
		"199.249.224.0/24",
		// Add more ranges as needed
	}

	for _, cidr := range torRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(net.ParseIP(ip)) {
			return true
		}
	}

	return false
}

// isSuspiciousUA checks for suspicious user agent patterns
func isSuspiciousUA(ua string) bool {
	suspiciousPatterns := []string{
		"curl", "wget", "python", "java", "go-http-client",
		"postman", "insomnia", "paw", "httpie",
	}

	uaLower := strings.ToLower(ua)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(uaLower, pattern) {
			return true
		}
	}
	return false
}

// LocationInfo represents geolocation data from IP address
type LocationInfo struct {
	IP       string `json:"ip"`
	Country  string `json:"countryCode"`
	Region   string `json:"regionName"`
	City     string `json:"city"`
	Timezone string `json:"timezone"`
	ISP      string `json:"isp"`
	Status   string `json:"status"`
}

// IPInfoResponse represents response from ipinfo.io
type IPInfoResponse struct {
	IP       string `json:"ip"`
	Country  string `json:"country"`
	Region   string `json:"region"`
	City     string `json:"city"`
	Timezone string `json:"timezone"`
	Org      string `json:"org"`
}

// IPAPIResponse represents response from ipapi.co
type IPAPIResponse struct {
	IP          string `json:"ip"`
	CountryCode string `json:"country_code"`
	Region      string `json:"region"`
	City        string `json:"city"`
	Timezone    string `json:"timezone"`
	Org         string `json:"org"`
	Error       bool   `json:"error,omitempty"`
}

// detectLocation detects geographical location from IP address with caching
func detectLocation(deviceInfo *DeviceInfo) {
	detectLocationWithCache(deviceInfo, nil)
}

// detectLocationWithCache detects geographical location from IP address with optional Redis cache
func detectLocationWithCache(deviceInfo *DeviceInfo, redisCache *cache.Redis) {
	// Skip location detection for private/local IPs

	if isPrivateIP(deviceInfo.IPAddress) {
		deviceInfo.Country = ""
		deviceInfo.Region = "Unknown"
		deviceInfo.City = "Unknown"
		deviceInfo.Timezone = ""
		deviceInfo.ISP = ""
		return
	}

	// Try to get location from cache first
	var location *LocationInfo
	if redisCache != nil {
		cachedLocation := getLocationFromCache(deviceInfo.IPAddress, redisCache)
		if cachedLocation != nil {
			location = cachedLocation
		}
	}

	// If not in cache, fetch from API with fallback providers
	if location == nil {
		location = getLocationWithFallback(deviceInfo.IPAddress)

		// Cache the result if we have Redis and a successful lookup
		if redisCache != nil && location != nil {
			cacheLocation(deviceInfo.IPAddress, location, redisCache)
		}
	}

	// Set device info fields
	if location != nil {
		deviceInfo.Country = location.Country
		deviceInfo.Region = location.Region
		deviceInfo.City = location.City
		deviceInfo.Timezone = location.Timezone
		deviceInfo.ISP = location.ISP
	} else {
		// Fallback values if API call fails
		deviceInfo.Country = "Unknown"
		deviceInfo.Region = "Unknown"
		deviceInfo.City = "Unknown"
		deviceInfo.Timezone = "Unknown"
		deviceInfo.ISP = "Unknown"
	}

	fmt.Println("Location Data: API -", location)
}

// getLocationFromIPAPI fetches location data from ip-api.com
func getLocationFromIPAPI(ip string) *LocationInfo {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Make request to ip-api.com
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,countryCode,region,regionName,city,lat,lon,timezone,isp", ip)
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	// Parse response
	var location LocationInfo
	if err := json.NewDecoder(resp.Body).Decode(&location); err != nil {
		return nil
	}

	fmt.Println("Location Data:", location)

	// Check if request was successful
	if location.Status != "success" {
		return nil
	}

	return &location
}

// getLocationWithFallback tries multiple providers in order with fallback
func getLocationWithFallback(ip string) *LocationInfo {
	// Provider priority order
	providers := []func(string) *LocationInfo{
		getLocationFromIPAPI,   // Primary: ip-api.com (free, reliable)
		getLocationFromIPInfo,  // Secondary: ipinfo.io (free tier)
		getLocationFromIPAPICO, // Tertiary: ipapi.co (free tier)
	}

	for i, provider := range providers {
		location := provider(ip)
		if location != nil {
			// Log which provider was used for monitoring
			if i > 0 {
				fmt.Printf("Location detection: Provider %d succeeded for IP %s\n", i+1, ip)
			}
			return location
		}
		fmt.Printf("Location detection: Provider %d failed for IP %s\n", i+1, ip)
	}

	fmt.Printf("Location detection: All providers failed for IP %s\n", ip)
	return nil
}

// getLocationFromIPInfo fetches location data from ipinfo.io
func getLocationFromIPInfo(ip string) *LocationInfo {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// ipinfo.io provides free tier without API key
	url := fmt.Sprintf("https://ipinfo.io/%s/json", ip)
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	var ipInfoResp IPInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&ipInfoResp); err != nil {
		return nil
	}

	// Convert to our standard LocationInfo format
	location := &LocationInfo{
		IP:       ipInfoResp.IP,
		Country:  ipInfoResp.Country, // Already 2-letter country code
		Region:   ipInfoResp.Region,  // Keep region name as-is
		City:     ipInfoResp.City,
		Timezone: ipInfoResp.Timezone,
		ISP:      ipInfoResp.Org,
		Status:   "success",
	}

	return location
}

// getLocationFromIPAPICO fetches location data from ipapi.co
func getLocationFromIPAPICO(ip string) *LocationInfo {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// ipapi.co provides free tier
	url := fmt.Sprintf("https://ipapi.co/%s/json/", ip)
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	var ipapiResp IPAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&ipapiResp); err != nil {
		return nil
	}

	// Check for API error
	if ipapiResp.Error {
		return nil
	}

	// Convert to our standard LocationInfo format following ip-api format
	location := &LocationInfo{
		IP:       ipapiResp.IP,
		Country:  ipapiResp.CountryCode, // Use 2-letter country code
		Region:   ipapiResp.Region,      // Keep region name as-is
		City:     ipapiResp.City,
		Timezone: ipapiResp.Timezone,
		ISP:      ipapiResp.Org,
		Status:   "success",
	}

	return location
}

// isPrivateIP checks if an IP address is private/local
func isPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return true // Invalid IP, treat as private
	}

	// Check for private IP ranges
	privateRanges := []string{
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"127.0.0.0/8",    // RFC3330
		"169.254.0.0/16", // RFC3927
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local
		"fe80::/10",      // IPv6 link local
	}

	for _, cidr := range privateRanges {
		_, subnet, err := net.ParseCIDR(cidr)
		if err == nil && subnet.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// getLocationFromCache attempts to retrieve location data from Redis cache
func getLocationFromCache(ip string, redisCache *cache.Redis) *LocationInfo {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("location:%s", ip)

	result, err := redisCache.Client.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil // Cache miss or error
	}

	var location LocationInfo

	fmt.Println("Cached Location Data:", result)

	if err := json.Unmarshal([]byte(result), &location); err != nil {
		return nil // Invalid cached data
	}

	return &location
}

// cacheLocation stores location data in Redis cache with expiration
func cacheLocation(ip string, location *LocationInfo, redisCache *cache.Redis) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("location:%s", ip)

	data, err := json.Marshal(location)
	if err != nil {
		return // Skip caching if marshal fails
	}

	// Cache for 24 hours (location data doesn't change frequently)
	redisCache.Client.Set(ctx, cacheKey, data, 24*time.Hour)
}

// DetectDeviceWithCache is a public function that allows passing Redis cache for location detection
func DetectDeviceWithCache(r *http.Request, redisCache *cache.Redis) *DeviceInfo {
	userAgent := r.Header.Get("User-Agent")
	ipAddress := GetClientIP(r)

	deviceInfo := &DeviceInfo{
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}

	// Parse user agent
	parseUserAgent(deviceInfo)

	// Detect location from IP with caching
	detectLocationWithCache(deviceInfo, redisCache)

	// Generate formatted device name
	deviceInfo.DeviceName = formatDeviceName(deviceInfo)

	// Detect security risks
	detectSecurityRisks(deviceInfo)

	// Generate device fingerprint
	deviceInfo.Fingerprint = generateFingerprint(deviceInfo)

	return deviceInfo
}

// generateFingerprint creates a unique fingerprint for the device
func generateFingerprint(info *DeviceInfo) string {
	// Create a fingerprint based on device characteristics
	fingerprint := fmt.Sprintf("%s|%s|%s|%s",
		info.OS,
		info.Browser,
		info.IPAddress,
		info.UserAgent)

	// Hash the fingerprint
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fingerprint)))
}

// Close closes the GeoIP database
func Close() error {
	// if d.geoIPDB != nil {
	// 	return d.geoIPDB.Close()
	// }
	return nil
}
