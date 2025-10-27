package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	ServerHost            string  `mapstructure:"SERVER_HOST" validate:"required"`
	ServerPort            int     `mapstructure:"SERVER_PORT" validate:"required,gte=1023,lte=65535"`
	DatabaseDSN           string  `mapstructure:"DB_DSN" validate:"required"`
	RedisPort             int     `mapstructure:"REDIS_PORT" validate:"required,gte=1023,lte=65535"`
	RedisDB               int     `mapstructure:"REDIS_DB" validate:"gte=0,lte=16"`
	JwtSecret             string  `mapstructure:"JWT_SECRET" validate:"required,min=32"`
	SmtpHost              string  `mapstructure:"SMTP_HOST" validate:"required"`
	SmtpPort              int     `mapstructure:"SMTP_PORT" validate:"required"`
	SmtpUsername          string  `mapstructure:"SMTP_USERNAME" validate:"required"`
	SmtpPassword          string  `mapstructure:"SMTP_PASSWORD" validate:"required"`
	RataLimitCapacity     float64 `mapstructure:"RATE_LIMITER_CAPACITY" validate:"required,gte=0"`
	RataLimitFillRate     float64 `mapstructure:"RATE_LIMITER_FILL_RATE" validate:"required,gte=0"`
	OTPLength             int     `mapstructure:"OTP_LENGTH" validate:"required,gte=0"`
	OTPExpiration         int     `mapstructure:"EXPIRATION" validate:"required,gte=0"`
	BcryptCost            int     `mapstructure:"BCRYPT_COST" validate:"required,gte=0"`
	GemeniModel           string  `mapstructure:"GEMENI_MODEL" validate:"required"`
	GemeniAPI             string  `mapstructure:"GEMENI_API" validate:"required"`
	WorkerCounts          int     `mapstructure:"NUM_WORKERS" validate:"required"`
	JobQueueSize          int     `mapstructure:"JOB_QUEUE_SIZE" validate:"required"`
	MaxAllowedSize        int     `mapstructure:"JSON_BODY_MAX_SIZE" validate:"required,gte=0"`
	FromEmail             string  `mapstructure:"FROM_EMAIL" validate:"required"`
	JwtISS                string  `mapstructure:"ISS" validate:"required"`
	LogFile               string  `mapstructure:"LOGGING_FILE" validate:"required"`
	ServerShutdownTimeout int     `mapstructure:"SERVER_SHUTDOWN_TIMEOUT" validate:"required,gte=0"`
}

func LoadConfigs(path string) (*Config, error) {

	viper.SetConfigFile(path)
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	var Cfg Config

	err = viper.Unmarshal(&Cfg)
	if err != nil {
		return nil, err
	}

	validate := validator.New()

	err = validate.Struct(Cfg)
	if err != nil {
		return nil, err
	}

	return &Cfg, nil

}
