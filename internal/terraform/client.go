package terraform

import (
	"context"
	"fmt"
	"io"
	"net/http"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/sirupsen/logrus"
)

type Client struct {
	client       *tfe.Client
	organization string
	logger       *logrus.Entry
}

type Workspace struct {
	ID                   string
	Name                 string
	Description          string
	CurrentStateVersion  string
	HasState             bool
}

type StateData struct {
	WorkspaceName string
	StateContent  []byte
	Version       int
	StateID       string
	Metadata      map[string]interface{}
}

// NewClient cria um novo client para o Terraform Cloud
func NewClient(token, organization string) (*Client, error) {
	config := &tfe.Config{
		Token: token,
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar client do Terraform Cloud: %w", err)
	}

	logger := logrus.WithFields(logrus.Fields{
		"component":    "terraform-client",
		"organization": organization,
	})

	return &Client{
		client:       client,
		organization: organization,
		logger:       logger,
	}, nil
}

// ListWorkspaces lista todos os workspaces da organização
func (c *Client) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	c.logger.Info("Listando workspaces")

	options := &tfe.WorkspaceListOptions{
		ListOptions: tfe.ListOptions{
			PageSize: 100,
		},
	}

	var allWorkspaces []Workspace

	for {
		workspaces, err := c.client.Workspaces.List(ctx, c.organization, options)
		if err != nil {
			return nil, fmt.Errorf("erro ao listar workspaces: %w", err)
		}

		for _, ws := range workspaces.Items {
			workspace := Workspace{
				ID:          ws.ID,
				Name:        ws.Name,
				Description: ws.Description,
				HasState:    ws.CurrentStateVersion != nil,
			}

			if ws.CurrentStateVersion != nil {
				workspace.CurrentStateVersion = ws.CurrentStateVersion.ID
			}

			allWorkspaces = append(allWorkspaces, workspace)
		}

		if workspaces.NextPage == 0 {
			break
		}
		options.PageNumber = workspaces.NextPage
	}

	c.logger.WithField("count", len(allWorkspaces)).Info("Workspaces listados com sucesso")
	return allWorkspaces, nil
}

// GetWorkspaceState obtém o estado atual de um workspace
func (c *Client) GetWorkspaceState(ctx context.Context, workspaceID string) (*StateData, error) {
	c.logger.WithField("workspace_id", workspaceID).Debug("Obtendo estado do workspace")

	// Primeiro, obter o workspace para verificar se tem estado
	workspace, err := c.client.Workspaces.ReadByID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler workspace %s: %w", workspaceID, err)
	}

	if workspace.CurrentStateVersion == nil {
		return nil, fmt.Errorf("workspace %s não possui estado atual", workspace.Name)
	}

	// Obter a versão do estado
	stateVersion, err := c.client.StateVersions.ReadCurrent(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler versão do estado para workspace %s: %w", workspace.Name, err)
	}

	// Download do conteúdo do estado
	stateURL := stateVersion.DownloadURL
	if stateURL == "" {
		return nil, fmt.Errorf("URL de download não disponível para o estado do workspace %s", workspace.Name)
	}

	// Fazer download do arquivo de estado
	resp, err := http.Get(stateURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer download do estado do workspace %s: %w", workspace.Name, err)
	}
	defer resp.Body.Close()

	stateContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler conteúdo do estado do workspace %s: %w", workspace.Name, err)
	}

	// Preparar metadata
	metadata := map[string]interface{}{
		"workspace_id":       workspace.ID,
		"workspace_name":     workspace.Name,
		"organization":       c.organization,
		"state_version_id":   stateVersion.ID,
		"serial":             stateVersion.Serial,
		"created_at":         stateVersion.CreatedAt.Format("2006-01-02T15:04:05Z"),
		"terraform_version":  stateVersion.TerraformVersion,
		"source":            "terraform_cloud",
	}

	if stateVersion.VCSCommitSHA != "" {
		metadata["vcs_commit_sha"] = stateVersion.VCSCommitSHA
	}

	stateData := &StateData{
		WorkspaceName: workspace.Name,
		StateContent:  stateContent,
		Version:       int(stateVersion.Serial),
		StateID:       stateVersion.ID,
		Metadata:      metadata,
	}

	c.logger.WithFields(logrus.Fields{
		"workspace_name": workspace.Name,
		"state_version":  stateVersion.Serial,
		"size_bytes":     len(stateContent),
	}).Debug("Estado obtido com sucesso")

	return stateData, nil
}

// GetWorkspaceByName obtém um workspace pelo nome
func (c *Client) GetWorkspaceByName(ctx context.Context, name string) (*Workspace, error) {
	c.logger.WithField("workspace_name", name).Debug("Buscando workspace por nome")

	workspace, err := c.client.Workspaces.Read(ctx, c.organization, name)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar workspace %s: %w", name, err)
	}

	ws := &Workspace{
		ID:          workspace.ID,
		Name:        workspace.Name,
		Description: workspace.Description,
		HasState:    workspace.CurrentStateVersion != nil,
	}

	if workspace.CurrentStateVersion != nil {
		ws.CurrentStateVersion = workspace.CurrentStateVersion.ID
	}

	return ws, nil
}

// ValidateConnection testa a conexão com o Terraform Cloud
func (c *Client) ValidateConnection(ctx context.Context) error {
	c.logger.Debug("Validando conexão com Terraform Cloud")

	_, err := c.client.Organizations.Read(ctx, c.organization)
	if err != nil {
		return fmt.Errorf("erro ao validar conexão com Terraform Cloud: %w", err)
	}

	c.logger.Info("Conexão com Terraform Cloud validada com sucesso")
	return nil
}