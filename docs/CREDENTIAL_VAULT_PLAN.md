# Credential Vault — Architecture & Implementation Plan

A secure credential vault built in **Go** that acts as an **MCP (Model Context Protocol) server**, allowing AI coding assistants to deploy to servers via SSH without ever exposing credentials in the LLM context.

---

## 1. Objective

Build a **Go** service that functions as a **secure vault for sensitive credentials**, accessible via **MCP** from any compatible AI assistant.

### Primary Use Case

1. User stores server credentials via CLI:
   - e.g. `Server PROD 1` → IP, user, password, port
2. When asking the AI assistant to "deploy to server X", the AI invokes the MCP `deploy` tool
3. The tool retrieves credentials **internally** (never exposed in plaintext to the LLM), connects via SSH, and runs the deploy command
4. Credentials **never** appear in the LLM context

---

## 2. Architecture

```
┌─────────────┐      stdio (MCP)      ┌──────────────────────┐
│   AI        │◄────────────────────►│  credential-vault    │
│   Assistant │   tools/requests      │  (MCP Server / Go)   │
└─────────────┘                       └──────────┬───────────┘
                                                 │
                          ┌──────────────────────┼───────────────────┐
                          │                      │                   │
                  ┌───────▼───────┐    ┌──────────▼─────────┐  ┌──────▼──────┐
                  │  Vault Store  │    │  Crypto Module     │  │  SSH Client │
                  │  (enc. JSON)  │    │  AES-256-GCM       │  │  (deploy)   │
                  └───────────────┘    │  Argon2id KDF      │  └─────────────┘
                                       └────────────────────┘
```

### Core Security Principle

> Credentials are decrypted **only inside the vault process**. The LLM never
> receives passwords, full IPs with credentials, or private keys in plaintext.
> The `deploy` tool receives only a server name + command, connects via SSH
> internally, and returns only the execution result.

---

## 3. Project Structure

```
credential-vault/
├── cmd/
│   └── credential-vault/        # Entry point (main.go)
├── internal/
│   ├── crypto/                  # AES-256-GCM + Argon2id encryption
│   │   └── crypto.go
│   ├── store/                   # Encrypted vault storage
│   │   └── store.go
│   ├── cli/                     # Command-line interface
│   │   └── cli.go
│   ├── mcpserver/               # MCP server (tools exposed to AI)
│   │   └── server.go
│   └── sshclient/               # SSH client for deploy
│       └── sshclient.go
├── tests/
│   └── integtest/               # Integration tests
├── go.mod
├── go.sum
└── docs/
    ├── CREDENTIAL_VAULT_PLAN.md  # This document
    └── MCP_CONFIG.md             # Configuration guide
```

---

## 4. Module Details

### 4.1 Crypto (`internal/crypto/crypto.go`)

- **Encryption algorithm**: AES-256-GCM (Authenticated Encryption)
- **KDF (Key Derivation)**: Argon2id
  - `memory=64MB`, `iterations=3`, `parallelism=2`
  - Random 16-byte salt per vault
- **Key functions**:
  - `DeriveKey(masterPassword, salt) → key`
  - `Encrypt(key, plaintext) → ciphertext`
  - `Decrypt(key, ciphertext) → plaintext`
- Libraries:
  - `crypto/aes`, `crypto/cipher` (Go stdlib)
  - `golang.org/x/crypto/argon2` (Argon2id)

### 4.2 Store (`internal/store/store.go`)

- **Vault format**: encrypted JSON file on disk
- **Logical structure** (before encryption):
  ```json
  {
    "servers": {
      "Server PROD 1": {
        "host": "192.168.1.10",
        "port": 22,
        "user": "root",
        "password": "super-secret-password",
        "private_key": "",
        "description": "Main production server"
      },
      "Server STAGING 1": {
        "host": "...",
        "port": 2222,
        "user": "deploy",
        "password": "...",
        "description": "Staging server"
      }
    }
  }
  ```
- **On-disk file format**:
  ```json
  {
    "version": 1,
    "kdf": {
      "algorithm": "argon2id",
      "salt": "<base64>",
      "memory": 65536,
      "iterations": 3,
      "parallelism": 2
    },
    "ciphertext": "<base64 of AES-256-GCM ciphertext>"
  }
  ```
- **Operations**: Load, Save, AddServer, GetServer, DeleteServer, ListServers

### 4.3 CLI (`internal/cli/cli.go`)

Available commands:

| Command | Description |
|---|---|
| `credential-vault init` | Initialize a new vault (set master password) |
| `credential-vault add <name>` | Add/update a server (interactive mode) |
| `credential-vault list` | List registered servers (never shows credentials) |
| `credential-vault delete <name>` | Remove a server |
| `credential-vault serve` | Start the MCP server (stdio) for AI assistants |

#### `credential-vault add` — interactive mode

```
$ credential-vault add "Server PROD 1"

Master password:
Host/IP: 192.168.1.10
SSH port [22]: 22
User: root
Password (leave empty to use private key):
Private key (paste content, Ctrl+D to finish):

Server 'Server PROD 1' added successfully!
```

### 4.4 MCP Server (`internal/mcpserver/server.go`)

Uses the official SDK: `github.com/modelcontextprotocol/go-sdk/mcp`

#### Exposed Tools

**1. `list_servers`**
- Description: "Lists server names and descriptions registered in the vault (no credentials exposed)"
- Input: none
- Output: list of names + descriptions (never host/user/password)

**2. `get_connection_info`**
- Description: "Returns safe connection info (host, port, user) without password"
- Input: `server_name` (string)
- Output: host, port, user, description — **no password, no private key**

**3. `deploy`**
- Description: "Connects via SSH to the server and executes a deploy command. Credentials are read from the vault and never exposed."
- Input:
  - `server_name` (string, required)
  - `command` (string, required) — command to execute
- Output: stdout + stderr of the remote command (no credentials)

**4. `ssh_exec`**
- Description: "Executes an arbitrary command via SSH on the specified server"
- Input:
  - `server_name` (string, required)
  - `command` (string, required)
- Output: stdout + stderr

---

### 4.5 SSH Client (`internal/sshclient/sshclient.go`)

- Uses `golang.org/x/crypto/ssh`
- Supports 2 auth modes:
  1. Password (`password`)
  2. Private key (`publickey`)
- **30s timeout** per connection
- Returns stdout + stderr separately

---

## 5. Security Flow

### 5.1 Storage

1. Master password → Argon2id → AES-256 key
2. Data → AES-256-GCM → ciphertext
3. `vault.json` contains only: salt + ciphertext + KDF parameters

### 5.2 MCP Access

1. AI assistant invokes `deploy` tool with `server_name` + `command`
2. MCP server reads `VAULT_PASSWORD` from environment
3. Decrypts vault in memory
4. Looks up credentials by name
5. Connects via SSH internally
6. Executes command
7. Returns only stdout/stderr to the LLM
8. **Credentials never leave the vault process**

### 5.3 Data Protection

- **Master password**: read via `VAULT_PASSWORD` environment variable
- **Vault file**: `~/.config/credential-vault/vault.json` (permission 0600)
- **No credentials are logged**
- **No secrets appear in tool output**
- `get_connection_info` returns host/port/user but **not** password/key

---

## 6. Environment Variables

| Variable | Description |
|---|---|
| `VAULT_PASSWORD` | Master password to decrypt the vault |
| `VAULT_PATH` | (Optional) Vault file path. Default: `~/.config/credential-vault/vault.json` |

---

## 7. Configuration in AI Assistants

After building the binary, add to your config file (e.g. `opencode.json`):

```json
{
  "mcp": {
    "credential-vault": {
      "type": "local",
      "command": ["/path/to/credential-vault", "serve"],
      "enabled": true,
      "environment": {
        "VAULT_PASSWORD": "your-master-password"
      }
    }
  }
}
```

See [MCP_CONFIG.md](MCP_CONFIG.md) for detailed instructions.

---

## 8. Technologies & Dependencies

| Dependency | Purpose |
|---|---|
| `github.com/modelcontextprotocol/go-sdk/mcp` | Official MCP Go SDK |
| `golang.org/x/crypto/argon2` | Argon2id KDF |
| `golang.org/x/crypto/ssh` | SSH client |
| `golang.org/x/term` | Terminal password reading |

---

## 9. Security Considerations

1. **Argon2id** is the PHC (Password Hashing Competition) winner — GPU/ASIC resistant
2. **AES-256-GCM** provides authenticity + confidentiality (tamper-proof)
3. The vault file does not contain the master password — only salt + ciphertext
4. The LLM never has access to credentials — the vault handles SSH internally
5. File permissions `0600` on the vault
6. Master password via env var, never via CLI argument (which appears in `ps`)
7. No credential logging under any circumstances

---

## 10. Usage — Full Flow

### Step 1: Build

```bash
go build -o credential-vault ./cmd/credential-vault
```

### Step 2: Initialize the vault

```bash
./credential-vault init
# Set the master password
```

### Step 3: Add servers

```bash
./credential-vault add "Server PROD 1"
# Fill in host, port, user, password
```

### Step 4: Configure in your AI assistant

See [MCP_CONFIG.md](MCP_CONFIG.md)

### Step 5: Use via AI assistant

```
User: "Deploy to Server PROD 1: run docker compose pull && docker compose up -d"

AI:
  → calls MCP tool "deploy" with server_name="Server PROD 1", command="docker compose pull && docker compose up -d"
  → vault connects via SSH internally
  → returns only the command result
```

---

## 11. Identified Risks

| Risk | Mitigation |
|---|---|
| Master password leak via `ps` | Password via env var, not CLI arg |
| Leak via logs | No credentials are logged |
| Vault file access | Permission 0600 + Argon2id makes brute-force infeasible |
| LLM extracting credentials | Tools never return password/key |
| Command injection via SSH | Command is passed directly to SSH exec; vault user is responsible |
| Vault corruption | Manual backup recommended; simple JSON format |

---

## 12. Rollback Strategy

- The vault is a single JSON file — keep a backup
- The tool does not modify external systems (only connects via SSH when requested)
- To revert: remove the MCP config from your assistant and delete the binary

---

<br>
<br>
<br>

---

# Credential Vault — Arquitetura & Plano de Implementação

Um cofre seguro de credenciais construído em **Go** que atua como um servidor **MCP (Model Context Protocol)**, permitindo que assistentes de IA de programação façam deploy em servidores via SSH sem nunca expor credenciais no contexto do LLM.

---

## 1. Objetivo

Construir um serviço em **Go** que funcione como um **cofre seguro de credenciais sensíveis**, acessível via **MCP** a partir de qualquer assistente de IA compatível.

### Caso de Uso Principal

1. O usuário armazena credenciais de servidores via CLI:
   - Ex: `Server PROD 1` → IP, usuário, senha, porta
2. Ao pedir ao assistente de IA para "fazer deploy no servidor X", a IA invoca a ferramenta MCP `deploy`
3. A ferramenta busca as credenciais **internamente** (sem expor em texto plano ao LLM), conecta via SSH e executa o deploy
4. As credenciais **nunca** aparecem no contexto do LLM

---

## 2. Arquitetura

```
┌─────────────┐      stdio (MCP)      ┌──────────────────────┐
│   Assistente│◄────────────────────►│  credential-vault    │
│   IA        │   tools/requests      │  (MCP Server / Go)   │
└─────────────┘                       └──────────┬───────────┘
                                                 │
                          ┌──────────────────────┼───────────────────┐
                          │                      │                   │
                  ┌───────▼───────┐    ┌──────────▼─────────┐  ┌──────▼──────┐
                  │  Vault Store  │    │  Crypto Module     │  │  SSH Client │
                  │  (JSON encr.) │    │  AES-256-GCM       │  │  (deploy)   │
                  └───────────────┘    │  Argon2id KDF      │  └─────────────┘
                                       └────────────────────┘
```

### Princípio de Segurança Central

> As credenciais são desencriptadas **apenas dentro do processo do vault**. O LLM
> nunca recebe senhas, IPs completos com credenciais, ou chaves privadas em texto
> plano. A ferramenta `deploy` recebe apenas um nome de servidor + comando, conecta
> via SSH internamente, e retorna apenas o resultado da execução.

---

## 3. Estrutura do Projeto

```
credential-vault/
├── cmd/
│   └── credential-vault/        # Entry point (main.go)
├── internal/
│   ├── crypto/                  # Criptografia AES-256-GCM + Argon2id
│   │   └── crypto.go
│   ├── store/                   # Armazenamento criptografado (cofre)
│   │   └── store.go
│   ├── cli/                     # Interface de linha de comando
│   │   └── cli.go
│   ├── mcpserver/               # Servidor MCP (ferramentas expostas à IA)
│   │   └── server.go
│   └── sshclient/               # Cliente SSH para deploy
│       └── sshclient.go
├── tests/
│   └── integtest/               # Testes de integração
├── go.mod
├── go.sum
└── docs/
    ├── CREDENTIAL_VAULT_PLAN.md  # Este documento
    └── MCP_CONFIG.md             # Guia de configuração
```

---

## 4. Módulos Detalhados

### 4.1 Crypto (`internal/crypto/crypto.go`)

- **Algoritmo de criptografia**: AES-256-GCM (Authenticated Encryption)
- **KDF (Key Derivation)**: Argon2id
  - `memory=64MB`, `iterations=3`, `parallelism=2`
  - Salt aleatório de 16 bytes por vault
- **Funções principais**:
  - `DeriveKey(masterPassword, salt) → key`
  - `Encrypt(key, plaintext) → ciphertext`
  - `Decrypt(key, ciphertext) → plaintext`
- Bibliotecas:
  - `crypto/aes`, `crypto/cipher` (Go stdlib)
  - `golang.org/x/crypto/argon2` (Argon2id)

### 4.2 Store (`internal/store/store.go`)

- **Formato do cofre**: arquivo JSON encriptado em disco
- **Estrutura lógica** (antes da encriptação):
  ```json
  {
    "servers": {
      "Server PROD 1": {
        "host": "192.168.1.10",
        "port": 22,
        "user": "root",
        "password": "senha-super-secreta",
        "private_key": "",
        "description": "Servidor de producao principal"
      },
      "Server STAGING 1": {
        "host": "...",
        "port": 2222,
        "user": "deploy",
        "password": "...",
        "description": "Servidor de homologacao"
      }
    }
  }
  ```
- **Formato do arquivo em disco**:
  ```json
  {
    "version": 1,
    "kdf": {
      "algorithm": "argon2id",
      "salt": "<base64>",
      "memory": 65536,
      "iterations": 3,
      "parallelism": 2
    },
    "ciphertext": "<base64 do AES-256-GCM>"
  }
  ```
- **Operações**: Load, Save, AddServer, GetServer, DeleteServer, ListServers

### 4.3 CLI (`internal/cli/cli.go`)

Comandos disponíveis:

| Comando | Descrição |
|---|---|
| `credential-vault init` | Inicializa um novo vault (define senha mestra) |
| `credential-vault add <name>` | Adiciona/atualiza um servidor (modo interativo) |
| `credential-vault list` | Lista servidores cadastrados (sem exibir credenciais) |
| `credential-vault delete <name>` | Remove um servidor |
| `credential-vault serve` | Inicia o servidor MCP (stdio) para assistentes de IA |

#### `credential-vault add` — modo interativo

```
$ credential-vault add "Server PROD 1"

Senha mestra:
Nome do host/IP: 192.168.1.10
Porta SSH [22]: 22
Usuario: root
Senha (deixe vazio se for usar chave privada):
Chave privada (cole o conteudo, Ctrl+D para terminar):

Servidor 'Server PROD 1' adicionado com sucesso!
```

### 4.4 Servidor MCP (`internal/mcpserver/server.go`)

Usa o SDK oficial: `github.com/modelcontextprotocol/go-sdk/mcp`

#### Ferramentas (Tools) expostas

**1. `list_servers`**
- Descrição: "Lista os nomes e descricoes dos servidores cadastrados no vault (sem expor credenciais)"
- Input: nenhum
- Output: lista de nomes + descricoes (nunca host/user/senha)

**2. `get_connection_info`**
- Descrição: "Retorna informacoes de conexao seguras (host, porta, usuario) sem a senha"
- Input: `server_name` (string)
- Output: host, porta, usuario, descricao — **sem senha e sem chave privada**

**3. `deploy`**
- Descrição: "Conecta via SSH ao servidor e executa um comando de deploy. As credenciais sao lidas do vault e nunca expostas."
- Input:
  - `server_name` (string, obrigatório)
  - `command` (string, obrigatório) — comando a executar
- Output: stdout + stderr do comando remoto (sem credenciais)

**4. `ssh_exec`**
- Descrição: "Executa um comando arbitrario via SSH no servidor especificado"
- Input:
  - `server_name` (string, obrigatório)
  - `command` (string, obrigatório)
- Output: stdout + stderr

---

### 4.5 Cliente SSH (`internal/sshclient/sshclient.go`)

- Usa `golang.org/x/crypto/ssh`
- Suporta 2 modos de autenticação:
  1. Senha (`password`)
  2. Chave privada (`publickey`)
- **Timeout** de 30s por conexão
- Retorna stdout + stderr separados

---

## 5. Fluxo de Segurança

### 5.1 Armazenamento

1. Senha mestra → Argon2id → chave AES-256
2. Dados → AES-256-GCM → ciphertext
3. `vault.json` contém apenas: salt + ciphertext + parâmetros KDF

### 5.2 Acesso via MCP

1. Assistente de IA invoca tool `deploy` com `server_name` + `command`
2. Servidor MCP lê `VAULT_PASSWORD` do ambiente
3. Desencripta o vault em memória
4. Busca credenciais por nome
5. Conecta via SSH internamente
6. Executa o comando
7. Retorna apenas stdout/stderr ao LLM
8. **Credenciais nunca saem do processo do vault**

### 5.3 Proteção de Dados

- **Senha mestra**: lida via variável de ambiente `VAULT_PASSWORD`
- **Arquivo do vault**: `~/.config/credential-vault/vault.json` (permissão 0600)
- **Nenhuma credencial é logada**
- **Nenhum secret aparece no output das tools**
- A ferramenta `get_connection_info` retorna host/port/user mas **não** senha/chave

---

## 6. Variáveis de Ambiente

| Variável | Descrição |
|---|---|
| `VAULT_PASSWORD` | Senha mestra para desencriptar o vault |
| `VAULT_PATH` | (Opcional) Caminho do arquivo vault. Padrão: `~/.config/credential-vault/vault.json` |

---

## 7. Configuração em Assistentes de IA

Após construir o binário, adicionar no arquivo de configuração (ex: `opencode.json`):

```json
{
  "mcp": {
    "credential-vault": {
      "type": "local",
      "command": ["/caminho/para/credential-vault", "serve"],
      "enabled": true,
      "environment": {
        "VAULT_PASSWORD": "sua-senha-mestra"
      }
    }
  }
}
```

Ver [MCP_CONFIG.md](MCP_CONFIG.md) para instruções detalhadas.

---

## 8. Tecnologias e Dependências

| Dependência | Uso |
|---|---|
| `github.com/modelcontextprotocol/go-sdk/mcp` | SDK oficial MCP |
| `golang.org/x/crypto/argon2` | Argon2id KDF |
| `golang.org/x/crypto/ssh` | Cliente SSH |
| `golang.org/x/term` | Leitura de senha sem ecoar no terminal |

---

## 9. Considerações de Segurança

1. **Argon2id** é o vencedor da PHC (Password Hashing Competition) — resistente a GPU/ASIC
2. **AES-256-GCM** fornece autenticidade + confidencialidade (tamper-proof)
3. O arquivo do vault não contém a senha mestra — apenas salt + ciphertext
4. O LLM nunca tem acesso às credenciais — o vault faz o SSH internamente
5. Permissões de arquivo `0600` no vault
6. Senha mestra via env var, nunca em argumento de linha de comando (que aparece no `ps`)
7. Sem logs de credenciais em nenhuma circunstância

---

## 10. Como Usar — Fluxo Completo

### Passo 1: Build

```bash
go build -o credential-vault ./cmd/credential-vault
```

### Passo 2: Inicializar o vault

```bash
./credential-vault init
# Define a senha mestra
```

### Passo 3: Adicionar servidores

```bash
./credential-vault add "Server PROD 1"
# Preenche host, porta, usuário, senha
```

### Passo 4: Configurar no assistente de IA

Ver [MCP_CONFIG.md](MCP_CONFIG.md)

### Passo 5: Usar via assistente de IA

```
Usuário: "Faça deploy no Server PROD 1, execute: docker compose pull && docker compose up -d"

IA:
  → chama tool MCP "deploy" com server_name="Server PROD 1", command="docker compose pull && docker compose up -d"
  → vault conecta via SSH internamente
  → retorna apenas o resultado do comando
```

---

## 11. Riscos Identificados

| Risco | Mitigação |
|---|---|
| Vazamento de senha mestra via `ps` | Senha via env var, não via CLI arg |
| Vazamento via logs | Nenhuma credencial é logada |
| Acesso ao arquivo do vault | Permissão 0600 + Argon2id torna brute-force inviável |
| LLM tentar extrair credenciais | Ferramentas nunca retornam senha/chave |
| Command injection no SSH | Comando é passado direto para SSH exec; usuário do vault é responsável |
| Vault corrompido | Backup manual recomendado; formato JSON simples |

---

## 12. Rollback Strategy

- O vault é um único arquivo JSON — basta manter um backup
- A ferramenta não modifica sistemas externos (apenas conecta via SSH quando solicitado)
- Para reverter: remover a configuração MCP do assistente e deletar o binário