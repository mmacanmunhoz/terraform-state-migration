package s3client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sirupsen/logrus"
)

type Client struct {
	s3Client *s3.Client
	bucket   string
	prefix   string
	logger   *logrus.Entry
}

type UploadOptions struct {
	Key         string
	Content     []byte
	ContentType string
	Metadata    map[string]string
}

// NewClient cria um novo client S3
func NewClient(region, bucket, prefix, profile string) (*Client, error) {
	var cfg aws.Config
	var err error
	
	if profile != "" {
		// Carregar configuração com perfil específico
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(region),
			config.WithSharedConfigProfile(profile),
		)
	} else {
		// Carregar configuração padrão
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(region),
		)
	}
	
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar configuração AWS: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	logger := logrus.WithFields(logrus.Fields{
		"component": "s3-client",
		"bucket":    bucket,
		"region":    region,
	})

	client := &Client{
		s3Client: s3Client,
		bucket:   bucket,
		prefix:   prefix,
		logger:   logger,
	}

	return client, nil
}

// ValidateConnection valida se a conexão com S3 está funcionando
func (c *Client) ValidateConnection(ctx context.Context) error {
	c.logger.Debug("Validando conexão com S3")

	_, err := c.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err != nil {
		return fmt.Errorf("erro ao validar acesso ao bucket S3 '%s': %w", c.bucket, err)
	}

	c.logger.Info("Conexão com S3 validada com sucesso")
	return nil
}

// UploadState faz upload de um arquivo de estado para S3
func (c *Client) UploadState(ctx context.Context, organization, workspaceName string, stateContent []byte, metadata map[string]interface{}) error {
	// Gerar chave do objeto S3
	stateKey := c.generateStateKey(organization, workspaceName, "terraform.tfstate")
	metadataKey := c.generateStateKey(organization, workspaceName, "metadata.json")

	c.logger.WithFields(logrus.Fields{
		"workspace":   workspaceName,
		"state_key":   stateKey,
		"size_bytes":  len(stateContent),
	}).Info("Fazendo upload do estado")

	// Upload do arquivo de estado
	err := c.uploadFile(ctx, UploadOptions{
		Key:         stateKey,
		Content:     stateContent,
		ContentType: "application/json",
		Metadata: map[string]string{
			"workspace":    workspaceName,
			"organization": organization,
			"file-type":    "terraform-state",
		},
	})
	if err != nil {
		return fmt.Errorf("erro ao fazer upload do estado do workspace %s: %w", workspaceName, err)
	}

	// Preparar e fazer upload dos metadados
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar metadados para workspace %s: %w", workspaceName, err)
	}

	err = c.uploadFile(ctx, UploadOptions{
		Key:         metadataKey,
		Content:     metadataJSON,
		ContentType: "application/json",
		Metadata: map[string]string{
			"workspace":    workspaceName,
			"organization": organization,
			"file-type":    "metadata",
		},
	})
	if err != nil {
		return fmt.Errorf("erro ao fazer upload dos metadados do workspace %s: %w", workspaceName, err)
	}

	c.logger.WithFields(logrus.Fields{
		"workspace":     workspaceName,
		"state_key":     stateKey,
		"metadata_key":  metadataKey,
	}).Info("Upload concluído com sucesso")

	return nil
}

// CheckStateExists verifica se o estado já existe no S3
func (c *Client) CheckStateExists(ctx context.Context, organization, workspaceName string) (bool, error) {
	stateKey := c.generateStateKey(organization, workspaceName, "terraform.tfstate")

	_, err := c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(stateKey),
	})
	if err != nil {
		// Verificar se é erro "não encontrado"
		var notFound *types.NoSuchKey
		var notFoundBucket *types.NotFound
		if errors.As(err, &notFound) || errors.As(err, &notFoundBucket) {
			return false, nil
		}
		return false, fmt.Errorf("erro ao verificar existência do estado: %w", err)
	}

	return true, nil
}

// uploadFile faz upload de um arquivo para S3
func (c *Client) uploadFile(ctx context.Context, options UploadOptions) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(options.Key),
		Body:        bytes.NewReader(options.Content),
		ContentType: aws.String(options.ContentType),
	}

	// Adicionar metadados se fornecidos
	if len(options.Metadata) > 0 {
		input.Metadata = options.Metadata
	}

	_, err := c.s3Client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("erro ao fazer upload para S3: %w", err)
	}

	return nil
}

// generateStateKey gera a chave S3 para um arquivo de estado
func (c *Client) generateStateKey(organization, workspaceName, filename string) string {
	// Estrutura: projeto/terraform.tfstate
	// Exemplo: arcotech-aws-budget-alert/terraform.tfstate
	return filepath.Join(workspaceName, filename)
}