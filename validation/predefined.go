package validation

// NewDatabaseValidator 创建数据库配置验证器
func NewDatabaseValidator() *StructuredValidator {
	validator := NewRuleValidator("Database Configuration Validator")

	// 基础连接配置验证
	validator.AddStringRules("database.host",
		"required",
		"hostname",
	)

	validator.AddStringRules("database.port",
		"required",
		"port",
	)

	validator.AddRule("database.username", Required("Database username cannot be empty"))
	validator.AddRule("database.password", Required("Database password cannot be empty"))

	// 数据库类型验证
	validator.AddStringRules("database.type",
		"required",
		"enum:mysql,postgresql,sqlite,mongodb",
	)

	// 数据库名验证
	validator.AddRule("database.database", Required("Database name cannot be empty"))

	// 连接数验证
	validator.AddStringRules("database.max_conns",
		"range:1,100",
	)

	// 超时验证
	validator.AddStringRules("database.timeout",
		"required",
	)

	return validator
}

// NewWebServerValidator 创建Web服务器配置验证器
func NewWebServerValidator() *StructuredValidator {
	validator := NewRuleValidator("Web Server Configuration Validator")

	// 主机配置
	validator.AddStringRules("server.host",
		"required",
		"hostname",
	)

	// 端口配置
	validator.AddStringRules("server.port",
		"required",
		"port",
	)

	// 运行模式
	validator.AddStringRules("server.mode",
		"enum:development,production,testing",
	)

	// 超时配置
	validator.AddStringRules("server.timeout",
		"required",
	)

	// SSL配置验证
	validator.AddStringRules("server.ssl.enabled",
		"boolean",
	)

	// SSL文件路径验证（当启用SSL时）
	validator.AddRule("server.ssl.cert_file", Required("SSL certificate file path cannot be empty"))
	validator.AddRule("server.ssl.key_file", Required("SSL private key file path cannot be empty"))

	return validator
}

// NewRedisValidator 创建Redis配置验证器
func NewRedisValidator() *StructuredValidator {
	validator := NewRuleValidator("Redis Configuration Validator")

	// 主机配置
	validator.AddStringRules("redis.host",
		"required",
		"hostname",
	)

	// 端口配置
	validator.AddStringRules("redis.port",
		"required",
		"port",
	)

	// 数据库索引验证
	validator.AddStringRules("redis.db",
		"range:0,15",
	)

	// 密码验证（可选）
	validator.AddStringRules("redis.password",
		"string",
	)

	// 地址列表验证
	validator.AddRule("redis.addresses", Required("Redis address list cannot be empty"))

	// 超时验证
	validator.AddStringRules("redis.timeout",
		"required",
	)

	return validator
}

// NewLogValidator 创建日志配置验证器
func NewLogValidator() *StructuredValidator {
	validator := NewRuleValidator("Log Configuration Validator")

	// 日志级别验证
	validator.AddStringRules("logging.level",
		"required",
		"enum:debug,info,warn,error,fatal",
	)

	// 日志格式验证
	validator.AddStringRules("logging.format",
		"enum:json,text",
	)

	// 日志路径验证
	validator.AddRule("logging.path", Required("Log output path cannot be empty"))

	return validator
}

// NewEmailValidator 创建邮件配置验证器
func NewEmailValidator() *StructuredValidator {
	validator := NewRuleValidator("Email Configuration Validator")

	// SMTP主机验证
	validator.AddStringRules("email.smtp.host",
		"required",
		"hostname",
	)

	// SMTP端口验证
	validator.AddStringRules("email.smtp.port",
		"required",
		"port",
	)

	// SMTP用户名验证
	validator.AddStringRules("email.smtp.username",
		"required",
		"email",
	)

	// SMTP密码验证
	validator.AddRule("email.smtp.password", Required("SMTP password cannot be empty"))

	// 发件人邮箱验证
	validator.AddStringRules("email.from",
		"required",
		"email",
	)

	return validator
}

// NewAPIValidator 创建API配置验证器
func NewAPIValidator() *StructuredValidator {
	validator := NewRuleValidator("API Configuration Validator")

	// API基础URL验证
	validator.AddStringRules("api.base_url",
		"required",
		"url",
	)

	// API超时验证
	validator.AddStringRules("api.timeout",
		"required",
		"range:1,300",
	)

	// 限流配置验证
	validator.AddStringRules("api.rate_limit.enabled",
		"boolean",
	)

	validator.AddStringRules("api.rate_limit.requests_per_minute",
		"range:1,10000",
	)

	// API密钥验证
	validator.AddStringRules("api.auth.api_key",
		"required",
	)

	// JWT配置验证
	validator.AddStringRules("api.auth.jwt.secret",
		"required",
	)

	validator.AddStringRules("api.auth.jwt.expires_in",
		"required",
	)

	return validator
}

// NewCommonValidator 创建通用预定义验证器
func NewCommonValidator() *CompositeValidator {
	composite := NewCompositeValidator("Common Configuration Validator")
	composite.AddValidator(NewDatabaseValidator())
	composite.AddValidator(NewWebServerValidator())
	composite.AddValidator(NewRedisValidator())
	composite.AddValidator(NewLogValidator())
	return composite
}

// NewMinimalValidator 创建最小化验证器
func NewMinimalValidator() *CompositeValidator {
	composite := NewCompositeValidator("Minimal Configuration Validator")
	composite.AddValidator(NewDatabaseValidator())
	composite.AddValidator(NewWebServerValidator())
	return composite
}
