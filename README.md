# Credential Vault

**A secure credential vault for AI coding assistants.**

Store server credentials (SSH) in an encrypted vault and let your AI assistant (via MCP) deploy to servers **without ever exposing passwords or private keys in the LLM context**.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.3.0-green.svg)](https://dolutech.com)

> Built by [Dolutech](https://dolutech.com) — secure infrastructure tools for developers.

---

## How It Works

```
You: "Deploy to Server PROD 1: run docker compose up -d"

AI assistant ──MCP──► credential-vault ──SSH──► Server PROD 1
                     (reads credentials        (executes command)
                      internally, never
                      exposes them)

AI assistant ◄────── returns only stdout/stderr (no credentials)
```

The AI never sees passwords, private keys, or the vault contents. The vault connects via SSH internally and returns only the command output.

---

## Key Features

- **Zero credential exposure** — passwords and private keys never appear in the LLM context
- **AES-256-GCM encryption** — authenticated encryption at rest
- **Argon2id key derivation** — resistant to GPU/ASIC brute-force attacks
- **MCP server** — integrates with any MCP-compatible AI assistant (opencode, Claude Code, Claude Desktop, Cursor, Windsurf, Zed, Continue, Cline, and more)
- **Simple CLI** — add, list, delete, and serve credentials
- **Cross-platform** — Linux (Arch, Debian/Ubuntu, RHEL/Fedora), macOS, Windows
- **Single binary** — no runtime dependencies, just compile and run
- **Multiple SSH auth methods** — password, private key (with optional passphrase), private key file path, SSH certificate

---

## Platform Support

| OS | Architecture | Status |
|---|---|---|
| Linux (Arch, Debian, Ubuntu, RHEL, Fedora, etc.) | amd64 | ✅ Supported |
| Linux (Arch, Debian, Ubuntu, RHEL, Fedora, etc.) | arm64 | ✅ Supported |
| macOS (Intel) | amd64 | ✅ Supported |
| macOS (Apple Silicon) | arm64 | ✅ Supported |
| Windows | amd64 | ✅ Supported |
| Windows | arm64 | ✅ Supported |

### Default Vault File Locations

| OS | Path |
|---|---|
| Linux | `$XDG_CONFIG_HOME/credential-vault/vault.json` (default: `~/.config/credential-vault/vault.json`) |
| macOS | `~/Library/Application Support/credential-vault/vault.json` |
| Windows | `%AppData%\credential-vault\vault.json` |

---

## Quick Start

### Install

**Option A — Build from source (all platforms):**

```bash
git clone https://github.com/dolutech/credential-vault.git
cd credential-vault
go build -o credential-vault ./cmd/credential-vault
```

**Option B — Install script (Linux & macOS):**

```bash
git clone https://github.com/dolutech/credential-vault.git
cd credential-vault
./scripts/install.sh
```

**Option C — Using Make (cross-platform build):**

```bash
make build      # build for current platform
make build-all  # build for all platforms
make release    # create release archives in dist/
```

### Initialize the vault

```bash
./credential-vault init
# Set your master password
```

### Add a server

```bash
./credential-vault add "Server PROD 1"
# Fill in host, port, user, password (interactive prompts)
```

### Configure in your AI assistant

Add the MCP server to your config (see [docs/MCP_CONFIG.md](docs/MCP_CONFIG.md)):

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

### Use it

```
You: "Deploy to Server PROD 1: run docker compose pull && docker compose up -d"

→ The AI calls the MCP "deploy" tool
→ The vault connects via SSH internally
→ Returns only the command output
```

---

## CLI Commands

| Command | Description |
|---|---|
| `credential-vault init` | Initialize a new vault (set master password) |
| `credential-vault add <name>` | Add or update a server (interactive mode) |
| `credential-vault list` | List registered servers (never shows credentials) |
| `credential-vault delete <name>` | Remove a server from the vault |
| `credential-vault serve` | Start the MCP server (stdio) for AI assistants |
| `credential-vault --version` | Show version |
| `credential-vault help` | Show help |

### Environment Variables

| Variable | Required | Description |
|---|---|---|
| `VAULT_PASSWORD` | Yes (for `serve`) | Master password to decrypt the vault |
| `VAULT_PATH` | No | Vault file path (default: OS-specific, see [Platform Support](#platform-support)) |

---

## MCP Tools

The vault exposes 4 tools to the AI assistant:

| Tool | Description | Exposes credentials? |
|---|---|:---:|
| `list_servers` | Lists server names + descriptions | No |
| `get_connection_info` | Returns host, port, user (no password) | No |
| `deploy` | SSH to server and run a deploy command | No |
| `ssh_exec` | SSH to server and run any command | No |

---

## Security

### What the AI NEVER sees

- Passwords
- Private keys
- Vault file contents
- Master password

### What the AI sees

- Server names and descriptions
- Host, port, and user (via `get_connection_info`)
- Command output (stdout/stderr)

### How credentials are protected

1. **At rest**: AES-256-GCM + Argon2id (64MB, 3 iterations)
2. **In transit**: Master password via environment variable (not CLI args, not in `ps`)
3. **From the LLM**: The vault connects via SSH internally — credentials never leave the process
4. **SSH host verification**: Server host keys verified against `~/.ssh/known_hosts`

---

## Project Structure

```
credential-vault/
├── cmd/credential-vault/          # Entry point
├── internal/
│   ├── crypto/                     # AES-256-GCM + Argon2id
│   ├── store/                      # Encrypted vault storage (cross-platform paths)
│   ├── cli/                        # CLI commands
│   ├── mcpserver/                  # MCP server (tools)
│   └── sshclient/                  # SSH client for deploy
├── tests/integtest/                # Integration tests
├── scripts/
│   └── install.sh                  # Install script (Linux & macOS)
├── docs/
│   ├── USAGE_GUIDE.md            # Practical step-by-step usage guide
│   ├── CREDENTIAL_VAULT_PLAN.md  # Architecture & implementation plan
│   └── MCP_CONFIG.md             # Configuration guide (all AI assistants)
├── Makefile                        # Build, test, release targets
├── LICENSE                         # MIT License
├── go.mod
└── go.sum
```

---

## Dependencies

| Dependency | Purpose |
|---|---|
| `github.com/modelcontextprotocol/go-sdk` | Official MCP Go SDK |
| `golang.org/x/crypto/argon2` | Argon2id key derivation |
| `golang.org/x/crypto/ssh` | SSH client |
| `golang.org/x/crypto/ssh/knownhosts` | SSH host key verification |
| `golang.org/x/term` | Hidden password input (cross-platform) |

---

## Development

```bash
# Build
make build

# Run tests
make test

# Run go vet
make vet

# Build for all platforms
make build-all

# Create release archives
make release
```

---

## Documentation

| Document | Description |
|---|---|
| [Usage Guide](docs/USAGE_GUIDE.md) | Practical step-by-step guide — install, add servers, deploy, daily usage examples, troubleshooting |
| [MCP Configuration Guide](docs/MCP_CONFIG.md) | How to configure in opencode, Claude Code, Claude Desktop, Cursor, Windsurf, Zed, Continue, Cline |
| [Architecture & Implementation Plan](docs/CREDENTIAL_VAULT_PLAN.md) | Technical architecture, security design, module details, data flow |

---

## License

MIT License — see [LICENSE](LICENSE)

Copyright (c) 2026 [Dolutech](https://dolutech.com)

---

## About

Built and maintained by [Dolutech](https://dolutech.com).

Visit our blog: [dolutech.com](https://dolutech.com)

---

<br>
<br>
<br>

---

# Credential Vault (Português)

**Um cofre seguro de credenciais para assistentes de IA de programação.**

Armazene credenciais de servidores (SSH) num cofre encriptado e permita que seu assistente de IA (via MCP) faça deploy em servidores **sem nunca expor senhas ou chaves privadas no contexto do LLM**.

[![Licença: MIT](https://img.shields.io/badge/Licença-MIT-blue.svg)](LICENSE)
[![Versão](https://img.shields.io/badge/versão-0.3.0-green.svg)](https://dolutech.com)

> Desenvolvido pela [Dolutech](https://dolutech.com) — ferramentas de infraestrutura segura para desenvolvedores.

---

## Como Funciona

```
Você: "Faça deploy no Server PROD 1: execute docker compose up -d"

Assistente IA ──MCP──► credential-vault ──SSH──► Server PROD 1
                       (lê as credenciais           (executa o comando)
                        internamente, nunca
                        as expõe)

Assistente IA ◄────── retorna apenas stdout/stderr (sem credenciais)
```

A IA nunca vê senhas, chaves privadas, ou o conteúdo do cofre. O cofre conecta via SSH internamente e retorna apenas o resultado do comando.

---

## Principais Recursos

- **Zero exposição de credenciais** — senhas e chaves privadas nunca aparecem no contexto do LLM
- **Criptografia AES-256-GCM** — criptografia autenticada em disco
- **Derivação de chave Argon2id** — resistente a ataques brute-force em GPU/ASIC
- **Servidor MCP** — integra com qualquer assistente de IA compatível com MCP (opencode, Claude Code, Claude Desktop, Cursor, Windsurf, Zed, Continue, Cline, e mais)
- **CLI simples** — adicionar, listar, remover e servir credenciais
- **Multiplataforma** — Linux (Arch, Debian/Ubuntu, RHEL/Fedora), macOS, Windows
- **Binário único** — sem dependências de runtime, basta compilar e executar
- **Múltiplos métodos de auth SSH** — senha, chave privada (com passphrase opcional), caminho de ficheiro de chave, certificado SSH

---

## Suporte de Plataformas

| SO | Arquitetura | Status |
|---|---|---|
| Linux (Arch, Debian, Ubuntu, RHEL, Fedora, etc.) | amd64 | ✅ Suportado |
| Linux (Arch, Debian, Ubuntu, RHEL, Fedora, etc.) | arm64 | ✅ Suportado |
| macOS (Intel) | amd64 | ✅ Suportado |
| macOS (Apple Silicon) | arm64 | ✅ Suportado |
| Windows | amd64 | ✅ Suportado |
| Windows | arm64 | ✅ Suportado |

### Localizações Padrão do Arquivo do Cofre

| SO | Caminho |
|---|---|
| Linux | `$XDG_CONFIG_HOME/credential-vault/vault.json` (padrão: `~/.config/credential-vault/vault.json`) |
| macOS | `~/Library/Application Support/credential-vault/vault.json` |
| Windows | `%AppData%\credential-vault\vault.json` |

---

## Início Rápido

### Instalar

**Opção A — Compilar do código-fonte (todas as plataformas):**

```bash
git clone https://github.com/dolutech/credential-vault.git
cd credential-vault
go build -o credential-vault ./cmd/credential-vault
```

**Opção B — Script de instalação (Linux & macOS):**

```bash
git clone https://github.com/dolutech/credential-vault.git
cd credential-vault
./scripts/install.sh
```

**Opção C — Usando Make (compilação multiplataforma):**

```bash
make build      # compilar para a plataforma atual
make build-all  # compilar para todas as plataformas
make release    # criar pacotes de release em dist/
```

### Inicializar o cofre

```bash
./credential-vault init
# Define a sua senha mestra
```

### Adicionar um servidor

```bash
./credential-vault add "Server PROD 1"
# Preenche host, porta, usuário, senha (prompts interativos)
```

### Configurar no seu assistente de IA

Adicione o servidor MCP na sua configuração (ver [docs/MCP_CONFIG.md](docs/MCP_CONFIG.md)):

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

### Usar

```
Você: "Faça deploy no Server PROD 1: execute docker compose pull && docker compose up -d"

→ A IA chama a ferramenta MCP "deploy"
→ O cofre conecta via SSH internamente
→ Retorna apenas o resultado do comando
```

---

## Comandos CLI

| Comando | Descrição |
|---|---|
| `credential-vault init` | Inicializa um novo cofre (define senha mestra) |
| `credential-vault add <nome>` | Adiciona ou atualiza um servidor (modo interativo) |
| `credential-vault list` | Lista servidores cadastrados (nunca exibe credenciais) |
| `credential-vault delete <nome>` | Remove um servidor do cofre |
| `credential-vault serve` | Inicia o servidor MCP (stdio) para assistentes de IA |
| `credential-vault --version` | Mostra a versão |
| `credential-vault help` | Mostra a ajuda |

### Variáveis de Ambiente

| Variável | Obrigatória | Descrição |
|---|---|---|
| `VAULT_PASSWORD` | Sim (para `serve`) | Senha mestra para desencriptar o cofre |
| `VAULT_PATH` | Não | Caminho do arquivo do cofre (padrão: específico do SO, ver [Suporte de Plataformas](#suporte-de-plataformas)) |

---

## Ferramentas MCP

O cofre expõe 4 ferramentas ao assistente de IA:

| Ferramenta | Descrição | Expõe credenciais? |
|---|---|:---:|
| `list_servers` | Lista nomes + descrições dos servidores | Não |
| `get_connection_info` | Retorna host, porta, usuário (sem senha) | Não |
| `deploy` | SSH ao servidor e executa um comando de deploy | Não |
| `ssh_exec` | SSH ao servidor e executa qualquer comando | Não |

---

## Segurança

### O que a IA NUNCA vê

- Senhas
- Chaves privadas
- Conteúdo do arquivo do cofre
- Senha mestra

### O que a IA vê

- Nomes e descrições dos servidores
- Host, porta e usuário (via `get_connection_info`)
- Resultado dos comandos (stdout/stderr)

### Como as credenciais são protegidas

1. **Em disco**: AES-256-GCM + Argon2id (64MB, 3 iterações)
2. **Em trânsito**: Senha mestra via variável de ambiente (não em argumentos CLI, não no `ps`)
3. **Do LLM**: O cofre conecta via SSH internamente — as credenciais nunca saem do processo
4. **Verificação de host SSH**: Chaves de host dos servidores verificadas contra `~/.ssh/known_hosts`

---

## Estrutura do Projeto

```
credential-vault/
├── cmd/credential-vault/          # Entry point
├── internal/
│   ├── crypto/                     # AES-256-GCM + Argon2id
│   ├── store/                      # Armazenamento encriptado (caminhos multiplataforma)
│   ├── cli/                        # Comandos CLI
│   ├── mcpserver/                  # Servidor MCP (ferramentas)
│   └── sshclient/                  # Cliente SSH para deploy
├── tests/integtest/                # Testes de integração
├── scripts/
│   └── install.sh                  # Script de instalação (Linux & macOS)
├── docs/
│   ├── USAGE_GUIDE.md            # Guia de utilização prático passo-a-passo
│   ├── CREDENTIAL_VAULT_PLAN.md  # Arquitetura & plano de implementação
│   └── MCP_CONFIG.md             # Guia de configuração (todos os assistentes)
├── Makefile                        # Targets de build, teste, release
├── LICENSE                         # Licença MIT
├── go.mod
└── go.sum
```

---

## Dependências

| Dependência | Propósito |
|---|---|
| `github.com/modelcontextprotocol/go-sdk` | SDK oficial MCP em Go |
| `golang.org/x/crypto/argon2` | Derivação de chave Argon2id |
| `golang.org/x/crypto/ssh` | Cliente SSH |
| `golang.org/x/crypto/ssh/knownhosts` | Verificação de chave de host SSH |
| `golang.org/x/term` | Entrada de senha oculta (multiplataforma) |

---

## Desenvolvimento

```bash
# Compilar
make build

# Rodar testes
make test

# Rodar go vet
make vet

# Compilar para todas as plataformas
make build-all

# Criar pacotes de release
make release
```

---

## Documentação

| Documento | Descrição |
|---|---|
| [Guia de Utilização](docs/USAGE_GUIDE.md) | Guia prático passo-a-passo — instalar, adicionar servidores, deploy, exemplos de uso diário, resolução de problemas |
| [Guia de Configuração MCP](docs/MCP_CONFIG.md) | Como configurar no opencode, Claude Code, Claude Desktop, Cursor, Windsurf, Zed, Continue, Cline |
| [Arquitetura & Plano de Implementação](docs/CREDENTIAL_VAULT_PLAN.md) | Arquitetura técnica, design de segurança, detalhes dos módulos, fluxo de dados |

---

## Licença

Licença MIT — ver [LICENSE](LICENSE)

Copyright (c) 2026 [Dolutech](https://dolutech.com)

---

## Sobre

Desenvolvido e mantido pela [Dolutech](https://dolutech.com).

Visite nosso blog: [dolutech.com](https://dolutech.com)