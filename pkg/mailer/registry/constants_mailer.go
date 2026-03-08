package registry

const (
	ProviderResend   = "resend"
	ProviderSendGrid = "sendgrid"
)

const (
	ProviderLabelNone       = "none"
	ProviderLabelFailover   = "failover"
	ProviderLabelPriority   = "priority"
	ProviderLabelValidation = "validation"
	ProviderLabelTemplate   = "template"
	UnknownProviderName     = "unknown"
)

const (
	ResendAPIURL   = "https://api.resend.com"
	SendGridAPIURL = "https://api.sendgrid.com"
)

const (
	PathResendEmails     = "/emails"
	PathResendAPIKeys    = "/api-keys"
	PathSendGridMailSend = "/v3/mail/send"
	PathSendGridScopes   = "/v3/scopes"
)

const (
	HeaderAuthorization = "Authorization"
	HeaderContentType   = "Content-Type"
	HeaderMessageID     = "X-Message-Id"
)

const (
	AuthBearerPrefix    = "Bearer "
	MIMEApplicationJSON = "application/json"
)

const (
	JSONFrom             = "from"
	JSONTo               = "to"
	JSONSubject          = "subject"
	JSONHTML             = "html"
	JSONText             = "text"
	JSONReplyTo          = "reply_to"
	JSONCC               = "cc"
	JSONBCC              = "bcc"
	JSONID               = "id"
	JSONEmail            = "email"
	JSONPersonalizations = "personalizations"
	JSONContent          = "content"
	JSONType             = "type"
	JSONValue            = "value"
)

const (
	MIMETextHTML  = "text/html"
	MIMETextPlain = "text/plain"
)

const (
	URLSchemeHTTP  = "http"
	URLSchemeHTTPS = "https"
)

const (
	TemplateNameVerifyEmail   = "verify-email"
	TemplateNamePasswordReset = "password-reset"
)

const (
	EmailVerificationExpiryHours = 24
	PasswordResetExpiryHours     = 1
)

const (
	HTTPStatusSuccessMin = 200
	HTTPStatusSuccessMax = 300
)

const (
	MessageSeparator       = "; "
	StrategySendFailedText = "send failed"
)

const (
	MsgFailedMarshalPayloadFmt = "failed to marshal payload: %v"
	MsgFailedCreateRequestFmt  = "failed to create request: %v"
	MsgRequestFailedFmt        = "request failed: %v"
	MsgFailedParseResponseFmt  = "failed to parse response: %v"
	MsgResendAPIErrorFmt       = "Resend API error: %d - %s"
	MsgSendGridAPIErrorFmt     = "SendGrid API error: %d - %s"
	MsgProviderErrorFmt        = "%s: %s"
)
