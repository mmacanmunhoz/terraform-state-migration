package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	TerraformCloud TerraformCloudConfig `mapstructure:"terraform_cloud"`
	AWS            AWSConfig            `mapstructure:"aws"`
	Migration      MigrationConfig      `mapstructure:"migration"`
	Logging        LoggingConfig        `mapstructure:"logging"`
}

type TerraformCloudConfig struct {
	Token        string `mapstructure:"token"`
	Organization string `mapstructure:"organization"`
}

type AWSConfig struct {
	Region  string `mapstructure:"region"`
	Bucket  string `mapstructure:"bucket"`
	Prefix  string `mapstructure:"prefix"`
	Profile string `mapstructure:"profile"`
}

type MigrationConfig struct {
	BatchSize         int `mapstructure:"batch_size"`
	ConcurrentUploads int `mapstructure:"concurrent_uploads"`
	RetryAttempts     int `mapstructure:"retry_attempts"`
}

type LoggingConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// LoadConfig carrega a configuração do arquivo config.yaml ou variáveis de ambiente
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("$HOME/.terraform-migrator")

	// Configurar variáveis de ambiente
	viper.SetEnvPrefix("TFC")
	viper.BindEnv("terraform_cloud.token", "TFC_TOKEN")
	viper.BindEnv("terraform_cloud.organization", "TFC_ORGANIZATION")
	viper.BindEnv("aws.region", "AWS_REGION")
	viper.BindEnv("aws.bucket", "S3_BUCKET")
	viper.BindEnv("aws.prefix", "S3_PREFIX")

	viper.AutomaticEnv()

	// Definir valores padrão
	viper.SetDefault("aws.region", "us-east-1")
	viper.SetDefault("aws.prefix", "terraform-states/")
	viper.SetDefault("migration.batch_size", 5)
	viper.SetDefault("migration.concurrent_uploads", 3)
	viper.SetDefault("migration.retry_attempts", 3)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.file", "migration.log")

	// Tentar ler o arquivo de configuração
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("erro ao ler arquivo de configuração: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("erro ao deserializar configuração: %w", err)
	}

	// Validar configurações obrigatórias
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate valida se todas as configurações obrigatórias estão presentes
func (c *Config) Validate() error {
	if c.TerraformCloud.Token == "" {
		return fmt.Errorf("token do Terraform Cloud é obrigatório")
	}

	if c.TerraformCloud.Organization == "" {
		return fmt.Errorf("organização do Terraform Cloud é obrigatória")
	}

	if c.AWS.Bucket == "" {
		return fmt.Errorf("bucket S3 é obrigatório")
	}

	if c.Migration.BatchSize <= 0 {
		return fmt.Errorf("batch_size deve ser maior que 0")
	}

	if c.Migration.ConcurrentUploads <= 0 {
		return fmt.Errorf("concurrent_uploads deve ser maior que 0")
	}

	return nil
}

// GetConfigPath retorna o caminho do arquivo de configuração sendo usado
func GetConfigPath() string {
	return viper.ConfigFileUsed()
}

// SaveConfig salva a configuração atual no arquivo
func (c *Config) SaveConfig() error {
	viper.Set("terraform_cloud", c.TerraformCloud)
	viper.Set("aws", c.AWS)
	viper.Set("migration", c.Migration)
	viper.Set("logging", c.Logging)

	return viper.WriteConfig()
}