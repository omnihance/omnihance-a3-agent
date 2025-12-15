package constants

const (
	ErrorCodeUnauthorized        = "UNAUTHORIZED"
	ErrorCodeInternalServerError = "INTERNAL_SERVER_ERROR"
	ErrorCodeSetupAlreadyDone    = "SETUP_ALREADY_DONE"
	ErrorCodeBadRequest          = "BAD_REQUEST"
	ErrorCodeNotFound            = "NOT_FOUND"
	ErrorCodePathIsDirectory     = "PATH_IS_DIRECTORY"
	ErrorCodeFileNotViewable     = "FILE_NOT_VIEWABLE"
	ErrorCodeFileReadError       = "FILE_READ_ERROR"
)

const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleUser       = "viewer"
)

const (
	UserStatusPending  = "pending"
	UserStatusActive   = "active"
	UserStatusInactive = "inactive"
	UserStatusDeleted  = "deleted"
	UserStatusBanned   = "banned"
)

const (
	SignInSuccessMessage        = "Signed in successfully"
	SignUpSuccessMessageActive  = "Account was successfully created. You can login now!"
	SignUpSuccessMessagePending = "Account was successfully created. Waiting for approval!"
)

const (
	CookieName = "omnihance_a3_agent_session"
	CookiePath = "/"
)

const (
	ContextKeyUserEmail = "user_email"
	ContextKeyUserRoles = "user_roles"
)
