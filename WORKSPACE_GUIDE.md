# Guia Prático: Seleção de Projetos e Workspaces

## ✅ Respostas para suas perguntas:

### 1. **Sim, você consegue passar lista de projetos específicos**

```bash
# Migrar projetos específicos
./migrator migrate --projects "projeto-web,projeto-api,projeto-db"

# Com dry-run para testar primeiro
./migrator migrate --projects "projeto-web,projeto-api" --dry-run

# Ajustando batch size para projetos específicos
./migrator migrate --projects "proj1,proj2,proj3" --batch-size 1
```

### 2. **Sim, tem opção de fazer TODOS os projetos**

```bash
# Migrar TODOS os workspaces (padrão)
./migrator migrate

# TODOS com dry-run
./migrator migrate --dry-run

# TODOS com batch customizado
./migrator migrate --batch-size 10
```

### 3. **Sim, workspaces sem estado são considerados e tratados**

O migrator automaticamente:
- ✅ **Identifica** workspaces sem estado do Terraform
- ✅ **Ignora** eles durante a migração
- ✅ **Informa** quais foram ignorados nos logs
- ✅ **Mostra estatísticas** de quantos têm/não têm estado

## 📋 Como ver quais workspaces têm estado

```bash
# Lista todos os workspaces mostrando seu status
./migrator list
```

**Exemplo de saída:**
```
📋 Workspaces encontrados na organização 'minha-org':

1. ✅ projeto-web COM ESTADO
   🔑 ID: ws-abc123
   📦 Versão do estado: sv-def456

2. ❌ projeto-temp SEM ESTADO
   🔑 ID: ws-ghi789

3. ✅ projeto-api COM ESTADO
   🔑 ID: ws-jkl012
   📦 Versão do estado: sv-mno345

📊 Resumo:
   • Total de workspaces: 3
   • Com estado (migráveis): 2
   • Sem estado (serão ignorados): 1
```

## 🎯 Cenários Práticos

### Cenário 1: Organização Mista (alguns com estado, outros sem)

```bash
# 1. Ver o que tem para migrar
./migrator list

# 2. Testar migração de todos
./migrator migrate --dry-run

# 3. Migrar apenas os que interessam
./migrator migrate --projects "prod-web,prod-api,staging-web"
```

### Cenário 2: Migração Gradual

```bash
# Fase 1: Ambientes de produção primeiro
./migrator migrate --projects "prod-web,prod-api,prod-db" --batch-size 1

# Fase 2: Ambientes de staging
./migrator migrate --projects "staging-web,staging-api" --batch-size 2

# Fase 3: Ambientes de desenvolvimento (todos os restantes)
./migrator migrate --batch-size 5
```

### Cenário 3: Troubleshooting de Projetos Específicos

```bash
# Debug de um projeto específico
./migrator migrate --projects "projeto-problematico" --log-level debug

# Retry de projetos que falharam
./migrator migrate --projects "proj1,proj2" --batch-size 1 --log-level info
```

## 📊 Logs Detalhados sobre Workspaces

Durante a migração, você verá logs como:

```
INFO[2026-01-12T10:30:00Z] Migração de projetos específicos projects="[projeto-web, projeto-api]"
INFO[2026-01-12T10:30:05Z] Análise de workspaces concluída 
    total_found=10 with_state=7 without_state=3 already_migrated=2 to_migrate=5
INFO[2026-01-12T10:30:05Z] Workspaces sem estado do Terraform (serão ignorados) 
    workspaces="[projeto-temp, projeto-vazio, projeto-novo]"
INFO[2026-01-12T10:30:05Z] Workspaces já migrados anteriormente (serão pulados) 
    workspaces="[projeto-antigo, projeto-backup]"
```

## ⚙️ Configurações Recomendadas por Cenário

### Para organizações com muitos workspaces vazios:
```yaml
migration:
  batch_size: 10        # Pode ser maior já que muitos serão ignorados
  concurrent_uploads: 3
  
logging:
  level: "info"         # Para ver quais são ignorados
```

### Para migração seletiva:
```yaml
migration:
  batch_size: 3         # Menor para controle preciso
  concurrent_uploads: 2
  
logging:
  level: "debug"        # Para troubleshooting detalhado
```

## 🚨 Casos Especiais

### Workspace não encontrado:
```bash
# Se especificar um workspace que não existe
./migrator migrate --projects "workspace-inexistente,workspace-real"
```
**Resultado**: O migrator avisa sobre o inexistente e continua com os válidos.

### Workspace sem permissão:
O migrator tenta acessar e reporta erros de permissão nos logs.

### Workspace já migrado:
```bash
# Se tentar migrar novamente
./migrator migrate --projects "ja-migrado"
```
**Resultado**: Detecta que já existe no S3 e pula automaticamente.

## 💡 Dicas Importantes

1. **Sempre use `list` primeiro** para entender o que você tem
2. **Sempre use `--dry-run`** antes da migração real
3. **Workspaces sem estado são normais** - muitos projetos começam vazios
4. **O migrator é idempotente** - pode executar múltiplas vezes sem problemas
5. **Logs são seus amigos** - use `--log-level debug` para troubleshooting