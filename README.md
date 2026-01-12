# Terraform Cloud to S3 Migrator

Um projeto em Go para migrar estados do Terraform Cloud para o Amazon S3 com controle de batch processing.

## ✨ Funcionalidades

- ✅ Autenticação segura com Terraform Cloud e AWS
- ✅ Listagem e download de estados do Terraform Cloud
- ✅ Upload de estados para S3 com estrutura organizacional
- ✅ Controle de batch processing configurável
- ✅ Logging detalhado e tratamento de erros
- ✅ Interface CLI amigável
- ✅ Retry automático em caso de falhas
- ✅ Modo dry-run para simulação
- ✅ Suporte a migração seletiva de projetos

## 🚀 Instalação Rápida

```bash
git clone <repository>
cd terraform-cloud-s3-migrator
make build
```

O binário estará disponível em `build/migrator`

## ⚙️ Configuração

### 1. Arquivo de Configuração

Copie o arquivo de exemplo e configure:

```bash
cp config.example.yaml config.yaml
```

Edite `config.yaml` com suas informações:

```yaml
terraform_cloud:
  token: "your-terraform-cloud-token"     # Token da API do Terraform Cloud
  organization: "your-organization"        # Nome da sua organização

aws:
  region: "us-east-1"                      # Região AWS
  bucket: "your-s3-bucket"                 # Bucket S3 de destino
  prefix: "terraform-states/"              # Prefixo para organização

migration:
  batch_size: 5                            # Projetos por batch (ajuste conforme necessário)
  concurrent_uploads: 3                    # Uploads simultâneos por batch
  retry_attempts: 3                        # Tentativas em caso de falha
  
logging:
  level: "info"                            # debug, info, warn, error
  file: "migration.log"                    # Arquivo de log
```

### 2. Variáveis de Ambiente (Alternativo)

```bash
export TFC_TOKEN="your-terraform-cloud-token"
export TFC_ORGANIZATION="your-organization"
export AWS_REGION="us-east-1"
export S3_BUCKET="your-bucket"
export S3_PREFIX="terraform-states/"
```

## 📋 Como Usar

### Listar Workspaces Disponíveis

```bash
./build/migrator list
```

### Simulação (Dry Run)

Antes de executar a migração real, teste com dry-run:

```bash
./build/migrator migrate --dry-run
```

### Migração Completa

```bash
./build/migrator migrate
```

### Migração com Batch Personalizado

```bash
# Processar 10 projetos por vez
./build/migrator migrate --batch-size 10
```

### Migração de Projetos Específicos

```bash
./build/migrator migrate --projects "workspace1,workspace2,workspace3"
```

### Migração com Logs Detalhados

```bash
./build/migrator migrate --log-level debug
```

## 🏗️ Estrutura no S3

Os estados são organizados de forma hierárquica:

```
s3://your-bucket/terraform-states/
├── your-organization/
│   ├── workspace1/
│   │   ├── terraform.tfstate        # Estado do Terraform
│   │   └── metadata.json           # Metadados (versão, data, etc.)
│   ├── workspace2/
│   │   ├── terraform.tfstate
│   │   └── metadata.json
│   └── workspace3/
│       ├── terraform.tfstate
│       └── metadata.json
```

## 🛠️ Comandos de Desenvolvimento

### Compilação

```bash
make build          # Compilação simples
make build-all      # Compilação para múltiplas plataformas
```

### Testes

```bash
make test           # Executar testes
make lint           # Executar linter
make format         # Formatar código
```

### Instalação

```bash
make install        # Instalar no sistema (/usr/local/bin)
make uninstall      # Remover do sistema
```

### Exemplos Rápidos

```bash
make run-example    # Listar workspaces
make dry-run        # Dry run da migração
make migrate-batch  # Migração com batch de 3
```

## 🔧 Configurações Avançadas

### Ajuste de Performance

- **batch_size**: Aumente para processar mais workspaces por vez, mas cuidado com rate limiting
- **concurrent_uploads**: Controla quantos uploads simultâneos por batch
- **retry_attempts**: Número de tentativas em caso de falha

### Recomendações

- Para organizações pequenas (< 20 workspaces): `batch_size: 5`, `concurrent_uploads: 2`
- Para organizações médias (20-100 workspaces): `batch_size: 10`, `concurrent_uploads: 3`
- Para organizações grandes (> 100 workspaces): `batch_size: 15`, `concurrent_uploads: 5`

## 🚨 Pré-requisitos

### Terraform Cloud

1. Token de API com permissões de leitura nos workspaces
2. Acesso à organização desejada

### AWS

1. Credenciais AWS configuradas (AWS CLI, IAM roles, ou variáveis de ambiente)
2. Permissões no bucket S3:
   - `s3:PutObject`
   - `s3:GetObject` 
   - `s3:ListBucket`
   - `s3:HeadObject`

## 🔍 Troubleshooting

### Problema: "Token do Terraform Cloud é obrigatório"

**Solução**: Configure o token no arquivo `config.yaml` ou na variável `TFC_TOKEN`

### Problema: "Bucket S3 é obrigatório"

**Solução**: Configure o bucket no arquivo `config.yaml` ou na variável `S3_BUCKET`

### Problema: Rate limiting

**Solução**: Diminua o `batch_size` e `concurrent_uploads` na configuração

### Problema: Falhas de upload

**Solução**: Verifique as permissões AWS e aumente `retry_attempts`

## 📊 Logs e Monitoramento

O migrator gera logs detalhados mostrando:

- Progresso dos batches
- Estados de cada workspace
- Estatísticas finais (sucessos/falhas)
- Taxa de sucesso
- Tempo total de execução

Exemplo de saída:

```
INFO[2026-01-12T10:30:00Z] Iniciando migração batch_size=5 concurrent_uploads=3 organization=my-org target_bucket=my-bucket
INFO[2026-01-12T10:30:05Z] Processando batch batch=1 progress=20.0% total_batches=5 batch_size=5
INFO[2026-01-12T10:30:10Z] Workspace migrado com sucesso workspace=my-workspace
INFO[2026-01-12T10:35:00Z] Migração finalizada duration=5m0s failed=0 mode=Migração successful=25 total=25
INFO[2026-01-12T10:35:00Z] Taxa de sucesso success_rate=100.0%
```

## 🤝 Contribuição

1. Fork o projeto
2. Crie uma branch para sua feature
3. Commit suas mudanças
4. Push para a branch
5. Abra um Pull Request

## 📝 Licença

Este projeto está sob a licença MIT. Veja o arquivo LICENSE para detalhes.

---

**⚡ Dica**: Sempre execute um `dry-run` antes da migração real para verificar quais workspaces serão processados!