package constants

// Error code prefixes by domain.
const (
	ErrorDomainValidation     = "VAL"
	ErrorDomainSearch         = "SEA"
	ErrorDomainSQLite         = "SQL"
	ErrorDomainLMDB           = "LMD"
	ErrorDomainTelegram       = "TG"
	ErrorDomainAdmin          = "ADM"
	ErrorDomainAuthorization  = "AUTH"
	ErrorDomainConfiguration  = "CFG"
	ErrorDomainTimeout        = "TMO"
	ErrorDomainNetwork        = "NET"
	ErrorDomainInternal       = "INT"
)

// Concrete error codes.
const (
	ErrCodeValidation     = "VAL_001"
	ErrCodeSearch         = "SEA_001"
	ErrCodeSearchNotFound = "SEA_002"
	ErrCodeSQLite         = "SQL_001"
	ErrCodeLMDB           = "LMD_001"
	ErrCodeTelegram       = "TG_001"
	ErrCodeAdmin          = "ADM_001"
	ErrCodeAuthorization  = "AUTH_001"
	ErrCodeForbidden      = "AUTH_002"
	ErrCodeConfiguration  = "CFG_001"
	ErrCodeTimeout        = "TMO_001"
	ErrCodeNetwork        = "NET_001"
	ErrCodeInternal       = "INT_001"
)
