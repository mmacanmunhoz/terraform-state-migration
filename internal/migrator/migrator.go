package migrator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"terraform-cloud-s3-migrator/internal/config"
	"terraform-cloud-s3-migrator/internal/s3client"
	"terraform-cloud-s3-migrator/internal/terraform"

	"github.com/sirupsen/logrus"
)

type Migrator struct {
	tfClient *terraform.Client
	s3Client *s3client.Client
	config   *config.Config
	logger   *logrus.Entry
}

type MigrationOptions struct {
	DryRun   bool
	Projects []string
}

type MigrationStats struct {
	Total       int
	Successful  int
	Failed      int
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
	FailedItems []FailedMigration
}

type FailedMigration struct {
	WorkspaceName string
	Error         string
}

// NewMigrator cria uma nova instância do migrator
func NewMigrator(cfg *config.Config) (*Migrator, error) {
	// Criar client do Terraform Cloud
	tfClient, err := terraform.NewClient(cfg.TerraformCloud.Token, cfg.TerraformCloud.Organization)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar client do Terraform Cloud: %w", err)
	}

	// Criar client do S3
	s3Client, err := s3client.NewClient(cfg.AWS.Region, cfg.AWS.Bucket, cfg.AWS.Prefix, cfg.AWS.Profile, cfg.AWS.AccountID)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar client do S3: %w", err)
	}

	logger := logrus.WithField("component", "migrator")

	return &Migrator{
		tfClient: tfClient,
		s3Client: s3Client,
		config:   cfg,
		logger:   logger,
	}, nil
}

// removeEnvironmentSuffix remove sufixos comuns de ambiente do nome do workspace
func (m *Migrator) removeEnvironmentSuffix(workspaceName string) string {
	// Lista de sufixos de ambiente comuns
	envSuffixes := []string{"-stg", "-prd", "-dev", "-prod", "-staging", "-production", "-test", "-qa", "-uat"}

	for _, suffix := range envSuffixes {
		if strings.HasSuffix(strings.ToLower(workspaceName), suffix) {
			cleanName := workspaceName[:len(workspaceName)-len(suffix)]
			m.logger.WithFields(logrus.Fields{
				"original_name":  workspaceName,
				"clean_name":     cleanName,
				"removed_suffix": suffix,
			}).Debug("Nome do workspace limpo para upload no S3")
			return cleanName
		}
	}

	// Se não encontrou nenhum sufixo conhecido, retorna o nome original
	return workspaceName
}

// ValidateConnections valida as conexões com Terraform Cloud e S3
func (m *Migrator) ValidateConnections() error {
	ctx := context.Background()

	m.logger.Info("Validando conexões...")

	// Validar Terraform Cloud
	if err := m.tfClient.ValidateConnection(ctx); err != nil {
		return fmt.Errorf("falha na validação do Terraform Cloud: %w", err)
	}

	// Validar S3
	if err := m.s3Client.ValidateConnection(ctx); err != nil {
		return fmt.Errorf("falha na validação do S3: %w", err)
	}

	m.logger.Info("Todas as conexões validadas com sucesso")
	return nil
}

// ListWorkspaces lista todos os workspaces disponíveis
func (m *Migrator) ListWorkspaces() ([]terraform.Workspace, error) {
	ctx := context.Background()

	if err := m.ValidateConnections(); err != nil {
		return nil, err
	}

	return m.tfClient.ListWorkspaces(ctx)
}

// Migrate executa a migração dos estados
func (m *Migrator) Migrate(options MigrationOptions) error {
	ctx := context.Background()

	// Validar conexões antes de iniciar
	if err := m.ValidateConnections(); err != nil {
		return err
	}

	stats := &MigrationStats{
		StartTime: time.Now(),
	}

	// Obter lista de workspaces para migrar
	workspaces, err := m.getWorkspacesToMigrate(ctx, options.Projects)
	if err != nil {
		return fmt.Errorf("erro ao obter lista de workspaces: %w", err)
	}

	stats.Total = len(workspaces)

	if stats.Total == 0 {
		m.logger.Warn("Nenhum workspace encontrado para migração")
		return nil
	}

	m.logger.WithFields(logrus.Fields{
		"total_workspaces": stats.Total,
		"batch_size":       m.config.Migration.BatchSize,
		"dry_run":          options.DryRun,
	}).Info("Iniciando migração")

	// Processar em batches
	err = m.processBatches(ctx, workspaces, options, stats)
	if err != nil {
		return err
	}

	// Calcular estatísticas finais
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	m.logFinalStats(stats, options.DryRun)

	if stats.Failed > 0 {
		return fmt.Errorf("migração concluída com %d falhas", stats.Failed)
	}

	return nil
}

// getWorkspacesToMigrate obtém a lista de workspaces para migrar
func (m *Migrator) getWorkspacesToMigrate(ctx context.Context, projectFilter []string) ([]terraform.Workspace, error) {
	var workspaces []terraform.Workspace
	var notFoundProjects []string

	if len(projectFilter) > 0 {
		// Migrar apenas projetos específicos
		m.logger.WithField("projects", projectFilter).Info("Migrando projetos específicos")
		for _, projectName := range projectFilter {
			workspace, err := m.tfClient.GetWorkspaceByName(ctx, projectName)
			if err != nil {
				m.logger.WithField("workspace", projectName).Warn("Workspace não encontrado")
				notFoundProjects = append(notFoundProjects, projectName)
				continue
			}
			workspaces = append(workspaces, *workspace)
		}

		if len(notFoundProjects) > 0 {
			m.logger.WithField("not_found", notFoundProjects).Warn("Alguns projetos especificados não foram encontrados")
		}
	} else {
		// Migrar todos os workspaces
		m.logger.Info("Migrando TODOS os workspaces da organização")
		allWorkspaces, err := m.tfClient.ListWorkspaces(ctx)
		if err != nil {
			return nil, err
		}
		workspaces = allWorkspaces
	}

	// Filtrar e contar workspaces por estado
	var workspacesWithState []terraform.Workspace
	var workspacesWithoutState []string
	var existingStates []string

	for _, ws := range workspaces {
		if !ws.HasState {
			m.logger.WithField("workspace", ws.Name).Debug("Workspace sem estado do Terraform, pulando")
			workspacesWithoutState = append(workspacesWithoutState, ws.Name)
			continue
		}

		// Verificar se já existe no S3 (usando nome limpo)
		cleanName := m.removeEnvironmentSuffix(ws.Name)
		exists, err := m.s3Client.CheckStateExists(ctx, m.config.TerraformCloud.Organization, cleanName)
		if err != nil {
			m.logger.WithError(err).WithField("workspace", ws.Name).Warn("Erro ao verificar existência no S3")
			// Continua mesmo com erro de verificação
		}

		if exists {
			m.logger.WithField("workspace", ws.Name).Debug("Estado já existe no S3, pulando")
			existingStates = append(existingStates, ws.Name)
			continue
		}

		workspacesWithState = append(workspacesWithState, ws)
	}

	// Log de resumo
	m.logger.WithFields(logrus.Fields{
		"total_found":      len(workspaces),
		"with_state":       len(workspacesWithState),
		"without_state":    len(workspacesWithoutState),
		"already_migrated": len(existingStates),
		"to_migrate":       len(workspacesWithState),
	}).Info("Análise de workspaces concluída")

	if len(workspacesWithoutState) > 0 {
		m.logger.WithField("workspaces", workspacesWithoutState).Info("Workspaces sem estado do Terraform (serão ignorados)")
	}

	if len(existingStates) > 0 {
		m.logger.WithField("workspaces", existingStates).Info("Workspaces já migrados anteriormente (serão pulados)")
	}

	return workspacesWithState, nil
}

// processBatches processa os workspaces em batches
func (m *Migrator) processBatches(ctx context.Context, workspaces []terraform.Workspace, options MigrationOptions, stats *MigrationStats) error {
	batchSize := m.config.Migration.BatchSize
	totalBatches := (len(workspaces) + batchSize - 1) / batchSize

	for i := 0; i < len(workspaces); i += batchSize {
		end := i + batchSize
		if end > len(workspaces) {
			end = len(workspaces)
		}

		batch := workspaces[i:end]
		batchNumber := (i / batchSize) + 1

		m.logger.WithFields(logrus.Fields{
			"batch":         batchNumber,
			"total_batches": totalBatches,
			"batch_size":    len(batch),
			"progress":      fmt.Sprintf("%.1f%%", float64(i)/float64(len(workspaces))*100),
		}).Info("Processando batch")

		err := m.processBatch(ctx, batch, options, stats)
		if err != nil {
			m.logger.WithError(err).Error("Erro ao processar batch")
			// Continuar com próximo batch em caso de erro
		}

		// Pequeno delay entre batches para evitar rate limiting
		if batchNumber < totalBatches {
			time.Sleep(1 * time.Second)
		}
	}

	return nil
}

// processBatch processa um batch de workspaces
func (m *Migrator) processBatch(ctx context.Context, batch []terraform.Workspace, options MigrationOptions, stats *MigrationStats) error {
	// Usar semáforo para controlar concorrência
	sem := make(chan struct{}, m.config.Migration.ConcurrentUploads)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, workspace := range batch {
		wg.Add(1)
		go func(ws terraform.Workspace) {
			defer wg.Done()

			// Adquirir semáforo
			sem <- struct{}{}
			defer func() { <-sem }()

			err := m.migrateWorkspace(ctx, ws, options.DryRun)

			mu.Lock()
			if err != nil {
				stats.Failed++
				stats.FailedItems = append(stats.FailedItems, FailedMigration{
					WorkspaceName: ws.Name,
					Error:         err.Error(),
				})
				m.logger.WithError(err).WithField("workspace", ws.Name).Error("Falha na migração do workspace")
			} else {
				stats.Successful++
				m.logger.WithField("workspace", ws.Name).Info("Workspace migrado com sucesso")
			}
			mu.Unlock()
		}(workspace)
	}

	wg.Wait()
	return nil
}

// migrateWorkspace migra um workspace específico
func (m *Migrator) migrateWorkspace(ctx context.Context, workspace terraform.Workspace, dryRun bool) error {
	logger := m.logger.WithField("workspace", workspace.Name)

	// Obter estado do Terraform Cloud
	stateData, err := m.tfClient.GetWorkspaceState(ctx, workspace.ID)
	if err != nil {
		return fmt.Errorf("erro ao obter estado: %w", err)
	}

	if dryRun {
		logger.WithField("state_size", len(stateData.StateContent)).Info("Dry run: estado seria migrado")
		return nil
	}

	// Obter nome limpo para upload no S3
	stateName := m.removeEnvironmentSuffix(workspace.Name)

	// Retry logic para upload
	var uploadErr error
	for attempt := 1; attempt <= m.config.Migration.RetryAttempts; attempt++ {
		uploadErr = m.s3Client.UploadState(
			ctx,
			m.config.TerraformCloud.Organization,
			stateName,
			stateData.StateContent,
			stateData.Metadata,
		)

		if uploadErr == nil {
			break
		}

		if attempt < m.config.Migration.RetryAttempts {
			delay := time.Duration(attempt) * time.Second
			logger.WithError(uploadErr).WithField("attempt", attempt).Warnf("Falha no upload, tentando novamente em %v", delay)
			time.Sleep(delay)
		}
	}

	if uploadErr != nil {
		return fmt.Errorf("erro ao fazer upload após %d tentativas: %w", m.config.Migration.RetryAttempts, uploadErr)
	}

	return nil
}

// logFinalStats registra as estatísticas finais da migração
func (m *Migrator) logFinalStats(stats *MigrationStats, dryRun bool) {
	mode := "Migração"
	if dryRun {
		mode = "Dry run"
	}

	m.logger.WithFields(logrus.Fields{
		"mode":       mode,
		"total":      stats.Total,
		"successful": stats.Successful,
		"failed":     stats.Failed,
		"duration":   stats.Duration.String(),
	}).Info("Migração finalizada")

	if len(stats.FailedItems) > 0 {
		m.logger.Error("Workspaces que falharam:")
		for _, failed := range stats.FailedItems {
			m.logger.WithFields(logrus.Fields{
				"workspace": failed.WorkspaceName,
				"error":     failed.Error,
			}).Error("Falha na migração")
		}
	}

	// Calcular taxa de sucesso
	if stats.Total > 0 {
		successRate := float64(stats.Successful) / float64(stats.Total) * 100
		m.logger.WithField("success_rate", fmt.Sprintf("%.1f%%", successRate)).Info("Taxa de sucesso")
	}
}
