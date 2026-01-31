package auth

type TokenType struct {
	ACCESS_TOKEN   string
	REFRESH_TOKEN  string
	RESET_PASSWORD string
}

var TOKEN_TYPE = TokenType{
	ACCESS_TOKEN:   "access_token",
	REFRESH_TOKEN:  "refresh_token",
	RESET_PASSWORD: "reset_password",
}
