package main

import (
	"fmt"
	"os"
	"strings"

	"terraform-cloud-s3-migrator/internal/config"
	"terraform-cloud-s3-migrator/internal/migrator"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	batchSize   int
	dryRun      bool
	projects    string
	logLevel    string
	appVersion  string = "dev" // Será definida durante o build
)

var rootCmd = &cobra.Command{
	Use:   "migrator",
	Short: "Terraform Cloud to S3 State Migrator",
	Long: `Uma ferramenta para migrar estados do Terraform Cloud para o Amazon S3
com controle de batch processing e configuração flexível.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Mostra a versão do aplicativo",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Terraform Cloud to S3 Migrator %s\n", appVersion)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista todos os workspaces disponíveis no Terraform Cloud",
	Long: `Lista todos os workspaces da organização configurada no Terraform Cloud.
Mostra quais workspaces têm estado (migráveis) e quais não têm (serão ignorados).

Exemplos:
  migrator list                    # Lista todos os workspaces
  migrator list --log-level debug  # Lista com logs detalhados`,
	RunE:  runList,
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Executa a migração dos estados do Terraform Cloud para S3",
	Long: `Executa a migração dos estados do Terraform Cloud para o Amazon S3.

Opções de migração:
  • TODOS os workspaces: Execute sem especificar projetos
  • Projetos específicos: Use a flag --projects com lista separada por vírgula
  • Simulação: Use --dry-run para testar sem fazer alterações

Workspaces sem estado do Terraform são automaticamente ignorados.
Workspaces já migrados anteriormente são pulados.

Exemplos:
  migrator migrate                                    # Migra TODOS os workspaces
  migrator migrate --dry-run                          # Simula a migração
  migrator migrate --projects \"prod,staging\"           # Migra projetos específicos
  migrator migrate --batch-size 10                    # Ajusta tamanho do batch
  migrator migrate --projects \"app1,app2\" --dry-run   # Simula migração específica`,
	RunE: runMigrate,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Flags globais
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "arquivo de configuração (padrão é config.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "nível de log (debug, info, warn, error)")

	// Flags para o comando migrate
	migrateCmd.Flags().IntVar(&batchSize, "batch-size", 0, "número de projetos a processar por vez")
	migrateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "apenas simula a migração sem executá-la")
	migrateCmd.Flags().StringVar(&projects, "projects", "", "lista de projetos específicos para migrar (separados por vírgula)")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(migrateCmd)
}

func initConfig() {
	// Configurar logrus
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		PadLevelText:  true,
	})
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	setupLogging(cfg)

	m, err := migrator.NewMigrator(cfg)
	if err != nil {
		return fmt.Errorf("erro ao criar migrator: %w", err)
	}

	workspaces, err := m.ListWorkspaces()
	if err != nil {
		return fmt.Errorf("erro ao listar workspaces: %w", err)
	}

	// Contar workspaces por categoria
	var withState, withoutState int
	for _, ws := range workspaces {
		if ws.HasState {
			withState++
		} else {
			withoutState++
		}
	}

	fmt.Printf("\n📋 Workspaces encontrados na organização '%s':\n\n", cfg.TerraformCloud.Organization)
	
	for i, ws := range workspaces {
		stateIcon := "❌"
		stateText := "SEM ESTADO"
		if ws.HasState {
			stateIcon = "✅"
			stateText = "COM ESTADO"
		}
		
		fmt.Printf("%d. %s %s %s\n", i+1, stateIcon, ws.Name, stateText)
		if ws.Description != "" {
			fmt.Printf("   📝 Descrição: %s\n", ws.Description)
		}
		fmt.Printf("   🔑 ID: %s\n", ws.ID)
		if ws.HasState {
			fmt.Printf("   📦 Versão do estado: %s\n", ws.CurrentStateVersion)
		}
		fmt.Println()
	}

	fmt.Printf("📊 Resumo:\n")
	fmt.Printf("   • Total de workspaces: %d\n", len(workspaces))
	fmt.Printf("   • Com estado (migráveis): %d\n", withState)
	fmt.Printf("   • Sem estado (serão ignorados): %d\n", withoutState)
	
	if withState > 0 {
		fmt.Printf("\n💡 Para migrar TODOS os workspaces com estado:\n")
		fmt.Printf("   ./migrator migrate\n\n")
		fmt.Printf("💡 Para migrar workspaces específicos:\n")
		fmt.Printf("   ./migrator migrate --projects \"workspace1,workspace2\"\n\n")
		fmt.Printf("💡 Para simular a migração primeiro:\n")
		fmt.Printf("   ./migrator migrate --dry-run\n")
	}
	
	return nil
}

func runMigrate(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("erro ao carregar configuração: %w", err)
	}

	setupLogging(cfg)

	// Override de configurações via flags
	if batchSize > 0 {
		cfg.Migration.BatchSize = batchSize
	}

	if logLevel != "" {
		cfg.Logging.Level = logLevel
	}

	m, err := migrator.NewMigrator(cfg)
	if err != nil {
		return fmt.Errorf("erro ao criar migrator: %w", err)
	}

	// Preparar lista de projetos específicos
	var projectList []string
	if projects != "" {
		projectList = strings.Split(projects, ",")
		for i, p := range projectList {
			projectList[i] = strings.TrimSpace(p)
		}
		logrus.WithField("projects", projectList).Info("Projetos específicos selecionados para migração")
	}

	options := migrator.MigrationOptions{
		DryRun:   dryRun,
		Projects: projectList,
	}

	if dryRun {
		logrus.Info("🧪 MODO DRY-RUN ativado - nenhuma alteração será feita")
		logrus.Info("Use este modo para testar a migração antes de executá-la")
	}

	logrus.WithFields(logrus.Fields{
		"batch_size":         cfg.Migration.BatchSize,
		"concurrent_uploads": cfg.Migration.ConcurrentUploads,
		"target_bucket":      cfg.AWS.Bucket,
		"organization":       cfg.TerraformCloud.Organization,
	}).Info("Iniciando migração")

	if err := m.Migrate(options); err != nil {
		return fmt.Errorf("erro durante a migração: %w", err)
	}

	logrus.Info("🎉 Migração concluída com sucesso!")
	return nil
}

func setupLogging(cfg *config.Config) {
	// Configurar nível de log
	if cfg.Logging.Level != "" {
		level, err := logrus.ParseLevel(cfg.Logging.Level)
		if err == nil {
			logrus.SetLevel(level)
		}
	}

	// Configurar arquivo de log se especificado
	if cfg.Logging.File != "" {
		file, err := os.OpenFile(cfg.Logging.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			logrus.SetOutput(file)
		}
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}