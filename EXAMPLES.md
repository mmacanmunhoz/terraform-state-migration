# Exemplos Práticos de Uso

## Cenários Comuns

### 1. Primeira Migração (Organizações Pequenas)

Para organizações com até 20 workspaces:

```bash
# 1. Configure o arquivo config.yaml
cp config.example.yaml config.yaml
# Edite config.yaml com seus tokens e informações

# 2. Teste a conexão listando workspaces
./build/migrator list

# 3. Execute um dry-run para verificar
./build/migrator migrate --dry-run

# 4. Execute a migração real
./build/migrator migrate --batch-size 3
```

### 2. Migração em Lotes (Organizações Grandes)

Para organizações com muitos workspaces:

```bash
# Migração gradual - comece devagar
./build/migrator migrate --batch-size 2 --log-level debug

# Se tudo correr bem, aumente o batch
./build/migrator migrate --batch-size 5

# Para organizações muito grandes
./build/migrator migrate --batch-size 10
```

### 3. Migração Seletiva

Migrando apenas workspaces específicos:

```bash
# Lista todos os workspaces primeiro
./build/migrator list

# Migra apenas os workspaces críticos
./build/migrator migrate --projects "prod-webapp,prod-database,prod-networking"

# Migra workspaces de desenvolvimento
./build/migrator migrate --projects "dev-env1,dev-env2,staging"
```

### 4. Recuperação de Falhas

Se a migração falhou parcialmente:

```bash
# Execute novamente - o migrator pula workspaces já migrados
./build/migrator migrate

# Com mais tentativas para workspaces problemáticos
./build/migrator migrate --log-level debug

# Verificar logs detalhados
tail -f migration.log
```

## Configurações por Cenário

### Configuração Conservadora (Recomendada para início)

```yaml
migration:
  batch_size: 3
  concurrent_uploads: 2
  retry_attempts: 5

logging:
  level: "info"
  file: "migration.log"
```

### Configuração Balanceada

```yaml
migration:
  batch_size: 5
  concurrent_uploads: 3
  retry_attempts: 3

logging:
  level: "info" 
  file: "migration.log"
```

### Configuração Agressiva (Para organizações grandes)

```yaml
migration:
  batch_size: 10
  concurrent_uploads: 5
  retry_attempts: 2

logging:
  level: "warn"  # Menos verboso
  file: "migration.log"
```

## Scripts de Automação

### Script de Migração Gradual

```bash
#!/bin/bash
# migrate_gradual.sh

echo "🚀 Iniciando migração gradual..."

# Fase 1: Teste
echo "📋 Listando workspaces..."
./build/migrator list

echo "🧪 Executando dry-run..."
./build/migrator migrate --dry-run --batch-size 2

read -p "Continuar com a migração real? (y/N): " confirm
if [[ $confirm == [yY] ]]; then
    # Fase 2: Migração conservadora
    echo "🔄 Iniciando migração (batch pequeno)..."
    ./build/migrator migrate --batch-size 2 --log-level info
    
    # Fase 3: Verificação
    echo "📊 Verificando logs..."
    tail -20 migration.log
    
    echo "✅ Migração concluída!"
else
    echo "❌ Migração cancelada."
fi
```

### Script de Monitoramento

```bash
#!/bin/bash
# monitor_migration.sh

echo "📊 Monitorando migração em tempo real..."
echo "Pressione Ctrl+C para parar"

while true; do
    clear
    echo "=== Status da Migração ==="
    echo "Data: $(date)"
    echo
    
    # Últimas linhas do log
    echo "📝 Últimos eventos:"
    tail -10 migration.log
    
    echo
    echo "🔄 Atualizando em 30 segundos..."
    sleep 30
done
```

## Troubleshooting Avançado

### Problema: Rate Limiting do Terraform Cloud

```bash
# Diminua drasticamente o batch size
./build/migrator migrate --batch-size 1 --log-level debug

# Ou adicione delays maiores modificando o código
```

### Problema: Workspaces Muito Grandes

```bash
# Execute com logs detalhados para identificar workspaces grandes
./build/migrator migrate --log-level debug

# Migre workspaces grandes individualmente
./build/migrator migrate --projects "large-workspace-1"
```

### Problema: Falhas de Rede AWS

```bash
# Aumente as tentativas de retry
# Modifique config.yaml:
migration:
  retry_attempts: 10

# Execute com timeout maior (pode requerer modificação do código)
```

## Integração com CI/CD

### GitHub Actions Example

```yaml
name: Migrate Terraform States

on:
  workflow_dispatch:
    inputs:
      dry_run:
        description: 'Execute dry run only'
        required: false
        default: 'true'
        type: boolean
      batch_size:
        description: 'Batch size'
        required: false
        default: '5'

jobs:
  migrate:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.21'
    
    - name: Build migrator
      run: make build
    
    - name: Configure AWS credentials
      uses: aws-actions/configure-aws-credentials@v2
      with:
        aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
        aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
        aws-region: us-east-1
    
    - name: Run migration
      env:
        TFC_TOKEN: ${{ secrets.TFC_TOKEN }}
        TFC_ORGANIZATION: ${{ secrets.TFC_ORGANIZATION }}
        S3_BUCKET: ${{ secrets.S3_BUCKET }}
      run: |
        if [ "${{ github.event.inputs.dry_run }}" == "true" ]; then
          ./build/migrator migrate --dry-run --batch-size ${{ github.event.inputs.batch_size }}
        else
          ./build/migrator migrate --batch-size ${{ github.event.inputs.batch_size }}
        fi
    
    - name: Upload logs
      uses: actions/upload-artifact@v3
      if: always()
      with:
        name: migration-logs
        path: migration.log
```

## Validação e Verificação

### Script de Validação Pós-Migração

```bash
#!/bin/bash
# validate_migration.sh

echo "🔍 Validando migração..."

# Verificar se bucket existe e tem acesso
aws s3 ls s3://your-bucket/terraform-states/ > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ Acesso ao bucket S3 confirmado"
else
    echo "❌ Problema de acesso ao bucket S3"
    exit 1
fi

# Contar workspaces no TFC
echo "📊 Contando workspaces..."
TFC_COUNT=$(./build/migrator list | grep -c "ID:")

# Contar estados no S3
S3_COUNT=$(aws s3 ls s3://your-bucket/terraform-states/your-org/ --recursive | grep "terraform.tfstate" | wc -l)

echo "📋 Workspaces no TFC: $TFC_COUNT"
echo "📁 Estados no S3: $S3_COUNT"

if [ "$TFC_COUNT" -eq "$S3_COUNT" ]; then
    echo "✅ Migração completa - todos os workspaces migrados!"
else
    echo "⚠️  Diferença encontrada - verifique os logs"
fi
```