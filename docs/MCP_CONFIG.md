# MCP Configuration Guide

This guide describes how to configure **credential-vault** as an MCP server in your AI coding assistant, allowing the LLM to deploy to servers without ever having access to credentials in plaintext.

---

## Prerequisites

1. Go 1.24+ installed
2. An MCP-compatible AI assistant — supported:
   - [opencode](https://opencode.ai)
   - [Claude Code](https://docs.anthropic.com/en/docs/claude-code)
   - [Claude Desktop](https://claude.ai/download)
   - [Cursor](https://cursor.com)
   - [Windsurf](https://codeium.com/windsurf)
   - [Zed](https://zed.dev)
   - [Continue](https://continue.dev)
   - [Cline](https://github.com/cline/cline)
   - Any other MCP-compatible client
3. Access to the servers where you want to deploy

---

## Step 1: Build the binary

```bash
git clone <repo-url> credential-vault
cd credential-vault
go build -o credential-vault ./cmd/credential-vault
```

The `credential-vault` binary will be created in the project root.

> **Tip**: For system-wide use, copy the binary to a directory in your PATH:
> ```bash
> sudo cp credential-vault /usr/local/bin/
> ```

---

## Step 2: Initialize the vault

The vault is the encrypted file where credentials are stored.

```bash
./credential-vault init
```

You will be prompted to set a **master password**. This password:
- Is used to derive the AES-256 key via Argon2id
- Is **NEVER** stored in any file
- Should be strong and memorized (or kept in a password manager)

**Default vault location**: `~/.config/credential-vault/vault.json`

> To use a custom path, set `VAULT_PATH`:
> ```bash
> VAULT_PATH=/custom/path/vault.json ./credential-vault init
> ```

---

## Step 3: Add servers

Add each server to the vault using the `add` command (interactive mode):

```bash
./credential-vault add "Server PROD 1"
```

The system will prompt:

```
Host/IP: 192.168.1.10
SSH port [22]: 22
User: root
Password (leave empty to use private key): my-super-secret-password
Description (optional): Main production server
```

### Using a private key instead of a password

Leave the "Password" field empty and paste the private key content:

```bash
./credential-vault add "Server PROD 2"
```

```
Host/IP: 10.0.0.20
SSH port [22]: 2222
User: deploy
Password (leave empty to use private key):    ← leave empty, press Enter
Private key (paste content, Ctrl+D to finish):
-----BEGIN OPENSSH PRIVATE KEY-----
...
-----END OPENSSH PRIVATE KEY-----
Description (optional): Secondary production server
```

### Verify registered servers

```bash
./credential-vault list
```

Output:
```
Registered servers (2):

  - Server PROD 1: Main production server
  - Server PROD 2: Secondary production server
```

> **Important**: The `list` command **never** displays credentials.

---

## Step 4: Configure in your AI assistant

> Replace `/path/to/credential-vault` with the actual path to your compiled binary.
> Replace `your-master-password` with the master password you set during `init`.

---

### A. opencode (global config)

Edit `~/.config/opencode/opencode.json`:

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

**Project-level**: Alternatively, create `opencode.json` in your project root:

```json
{
  "mcp": {
    "credential-vault": {
      "type": "local",
      "command": ["/path/to/credential-vault", "serve"],
      "enabled": true,
      "environment": {
        "VAULT_PASSWORD": "your-master-password",
        "VAULT_PATH": "/path/to/vault.json"
      }
    }
  }
}
```

---

### B. Claude Code (CLI)

Add the MCP server via the CLI:

```bash
claude mcp add credential-vault -- /path/to/credential-vault serve
```

Then set the environment variable before launching Claude Code:

```bash
export VAULT_PASSWORD="your-master-password"
claude
```

Or add to your `~/.bashrc` / `~/.zshrc`:

```bash
export VAULT_PASSWORD="your-master-password"
```

---

### C. Claude Desktop

Edit `claude_desktop_config.json`:

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "credential-vault": {
      "command": "/path/to/credential-vault",
      "args": ["serve"],
      "env": {
        "VAULT_PASSWORD": "your-master-password"
      }
    }
  }
}
```

---

### D. Cursor

Cursor supports MCP servers via its settings.

**Via Cursor Settings UI**:
1. Open Cursor → Settings → Features → MCP
2. Click "Add new MCP server"
3. Name: `credential-vault`
4. Type: `stdio`
5. Command: `/path/to/credential-vault`
6. Args: `serve`
7. Env: `VAULT_PASSWORD=your-master-password`

**Via config file** — edit `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "credential-vault": {
      "command": "/path/to/credential-vault",
      "args": ["serve"],
      "env": {
        "VAULT_PASSWORD": "your-master-password"
      }
    }
  }
}
```

---

### E. Windsurf (Codeium)

Edit `~/.codeium/windsurf/mcp_config.json`:

```json
{
  "mcpServers": {
    "credential-vault": {
      "command": "/path/to/credential-vault",
      "args": ["serve"],
      "env": {
        "VAULT_PASSWORD": "your-master-password"
      }
    }
  }
}
```

---

### F. Zed

Edit `~/.config/zed/settings.json` and add under `context_servers`:

```json
{
  "context_servers": {
    "credential-vault": {
      "command": {
        "path": "/path/to/credential-vault",
        "args": ["serve"],
        "env": {
          "VAULT_PASSWORD": "your-master-password"
        }
      }
    }
  }
}
```

---

### G. Continue (VS Code / JetBrains)

Edit `~/.continue/config.json` and add under `mcpServers`:

```json
{
  "mcpServers": {
    "credential-vault": {
      "command": "/path/to/credential-vault",
      "args": ["serve"],
      "env": {
        "VAULT_PASSWORD": "your-master-password"
      }
    }
  }
}
```

---

### H. Cline (VS Code)

Cline uses the same format as Claude Desktop. Edit `~/Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json` (or the equivalent path on your OS):

```json
{
  "mcpServers": {
    "credential-vault": {
      "command": "/path/to/credential-vault",
      "args": ["serve"],
      "env": {
        "VAULT_PASSWORD": "your-master-password"
      }
    }
  }
}
```

---

### I. Using a system environment variable (more secure)

Instead of putting the password in the config file, set it in your shell profile (`~/.bashrc`, `~/.zshrc`):

```bash
export VAULT_PASSWORD="your-master-password"
```

Then in the MCP config, omit the `environment` / `env` block:

```json
{
  "mcp": {
    "credential-vault": {
      "type": "local",
      "command": ["/path/to/credential-vault", "serve"],
      "enabled": true
    }
  }
}
```

---

### Environment Variables

| Variable | Required | Description |
|---|---|---|
| `VAULT_PASSWORD` | **Yes** (for `serve`) | Master password to decrypt the vault |
| `VAULT_PATH` | No | Vault file path (default: `~/.config/credential-vault/vault.json`) |

---

## Step 5: Use via your AI assistant

After configuring, restart your AI assistant. The LLM now has access to the following tools:

### Tool: `list_servers`

Lists registered servers (names + descriptions, no credentials).

**Example:**
```
You: "Which servers are available for deploy?"

AI: [calls list_servers]
    "You have 2 servers registered:
     - Server PROD 1: Main production server
     - Server PROD 2: Secondary production server"
```

### Tool: `get_connection_info`

Returns host, port, and user of a server (no password/key).

**Example:**
```
You: "What's the address of Server PROD 1?"

AI: [calls get_connection_info with server_name="Server PROD 1"]
    "Server PROD 1: root@192.168.1.10:22"
```

### Tool: `deploy`

Connects via SSH to the server and executes a deploy command.
Credentials are read internally from the vault and **never** appear in the LLM context.

**Example:**
```
You: "Deploy to Server PROD 1: run docker compose pull && docker compose up -d"

AI: [calls deploy with server_name="Server PROD 1",
     command="docker compose pull && docker compose up -d"]

    "Deploy executed on Server PROD 1:
     stdout: Pulling web... done
             Starting web... done
     exit code: 0"
```

### Tool: `ssh_exec`

Executes an arbitrary command via SSH on the server.

**Example:**
```
You: "Check nginx status on Server PROD 1"

AI: [calls ssh_exec with server_name="Server PROD 1",
     command="systemctl status nginx"]

    "Command executed on Server PROD 1:
     stdout: nginx.service - The nginx HTTP and reverse proxy server
              Active: active (running)
     exit code: 0"
```

---

## Server Lifecycle

The MCP server uses **stdio** transport — it does NOT open a network port.

| Situation | What happens |
|---|---|
| You open your AI assistant | credential-vault **starts automatically** (as a subprocess) |
| You close your AI assistant | credential-vault **stops automatically** |
| You don't want to use the vault | Just don't call the MCP tools — it stays idle, no resource usage |
| Want to temporarily disable | Set `"enabled": false` in the config |

You **do NOT** need to:
- Start it manually (`./credential-vault serve &`)
- Kill the process manually
- Set up a systemd service

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

1. **At rest**: AES-256-GCM + Argon2id (authenticated encryption)
2. **In transit**: Master password via environment variable (not in `ps`)
3. **From the LLM**: The vault connects via SSH internally — credentials never leave the process

---

## Server Management

### Remove a server

```bash
./credential-vault delete "Server PROD 2"
```

### Update a server

Use the `add` command with the same name (overwrites):

```bash
./credential-vault add "Server PROD 1"
```

### Backup the vault

The vault is a single encrypted JSON file. To back up:

```bash
cp ~/.config/credential-vault/vault.json /backup/location/vault-backup-$(date +%Y%m%d).json
```

> **Remember**: Without the master password, the backup is useless. Keep the password
> separately in a password manager.

---

## Troubleshooting

### Error: "vault not found"

```
Error: vault not found — run 'credential-vault init' first
```

**Solution**: Run `./credential-vault init` to create the vault.

### Error: "VAULT_PASSWORD not set"

```
Error: VAULT_PASSWORD not set — the MCP server requires the password via environment variable
```

**Solution**: Make sure `VAULT_PASSWORD` is set in the `environment` block of the MCP config.

### Error: "failed to decrypt (wrong password or corrupted data)"

The master password is incorrect or the vault file is corrupted.

**Solution**:
1. Check that the password is correct
2. If the vault is corrupted, restore from backup with the correct password

### AI assistant doesn't see the tools

1. Verify the binary path is correct in the config
2. Verify the binary has execute permission: `chmod +x credential-vault`
3. Restart your AI assistant after changing the config

---

## Quick Summary

```bash
# 1. Build
go build -o credential-vault ./cmd/credential-vault

# 2. Init vault
./credential-vault init

# 3. Add server
./credential-vault add "Server PROD 1"

# 4. Configure in your AI assistant (see above)

# 5. Use: "Deploy to Server PROD 1: run docker compose up -d"
```

---

<br>
<br>
<br>

---

# Guia de Configuração MCP

Este guia descreve como configurar o **credential-vault** como um servidor MCP no seu assistente de IA de programação, permitindo que o LLM faça deploy em servidores sem nunca ter acesso às credenciais em texto plano.

---

## Pré-requisitos

1. Go 1.24+ instalado
2. Um assistente de IA compatível com MCP — suportados:
   - [opencode](https://opencode.ai)
   - [Claude Code](https://docs.anthropic.com/en/docs/claude-code)
   - [Claude Desktop](https://claude.ai/download)
   - [Cursor](https://cursor.com)
   - [Windsurf](https://codeium.com/windsurf)
   - [Zed](https://zed.dev)
   - [Continue](https://continue.dev)
   - [Cline](https://github.com/cline/cline)
   - Qualquer outro cliente compatível com MCP
3. Acesso aos servidores onde pretende fazer deploy

---

## Passo 1: Compilar o binário

```bash
git clone <repo-url> credential-vault
cd credential-vault
go build -o credential-vault ./cmd/credential-vault
```

O binário `credential-vault` será criado na raiz do projeto.

> **Dica**: Para uso sistema-wide, copie o binário para um diretório no PATH:
> ```bash
> sudo cp credential-vault /usr/local/bin/
> ```

---

## Passo 2: Inicializar o vault

O vault é o arquivo criptografado onde as credenciais ficam armazenadas.

```bash
./credential-vault init
```

Você será solicitado a definir uma **senha mestra**. Esta senha:
- É usada para derivar a chave AES-256 via Argon2id
- **NUNCA** é armazenada em nenhum arquivo
- Deve ser forte e memorizada (ou guardada em um gerenciador de senhas)

**Local padrão do vault**: `~/.config/credential-vault/vault.json`

> Para usar um caminho personalizado, defina `VAULT_PATH`:
> ```bash
> VAULT_PATH=/caminho/personalizado/vault.json ./credential-vault init
> ```

---

## Passo 3: Adicionar servidores

Adicione cada servidor ao vault usando o comando `add` (modo interativo):

```bash
./credential-vault add "Server PROD 1"
```

O sistema solicitará:

```
Nome do host/IP: 192.168.1.10
Porta SSH [22]: 22
Usuário: root
Senha (deixe vazio se for usar chave privada): minha-senha-super-secreta
Descrição (opcional): Servidor de produção principal
```

### Para usar chave privada em vez de senha

Deixe o campo "Senha" vazio e cole o conteúdo da chave privada:

```bash
./credential-vault add "Server PROD 2"
```

```
Nome do host/IP: 10.0.0.20
Porta SSH [22]: 2222
Usuário: deploy
Senha (deixe vazio se for usar chave privada):    ← deixe vazio e dê Enter
Cole o conteúdo da chave privada (Ctrl+D para terminar):
-----BEGIN OPENSSH PRIVATE KEY-----
...
-----END OPENSSH PRIVATE KEY-----
Descrição (opcional): Servidor de produção secundário
```

### Verificar servidores cadastrados

```bash
./credential-vault list
```

Saída:
```
Servidores cadastrados (2):

  - Server PROD 1: Servidor de produção principal
  - Server PROD 2: Servidor de produção secundário
```

> **Importante**: O comando `list` **nunca** exibe credenciais.

---

## Passo 4: Configurar no assistente de IA

> Substitua `/caminho/para/credential-vault` pelo caminho real do seu binário compilado.
> Substitua `sua-senha-mestra` pela senha mestra definida durante o `init`.

---

### A. opencode (configuração global)

Edite `~/.config/opencode/opencode.json`:

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

**Por projeto**: Alternativamente, crie `opencode.json` na raiz do seu projeto:

```json
{
  "mcp": {
    "credential-vault": {
      "type": "local",
      "command": ["/caminho/para/credential-vault", "serve"],
      "enabled": true,
      "environment": {
        "VAULT_PASSWORD": "sua-senha-mestra",
        "VAULT_PATH": "/caminho/para/vault.json"
      }
    }
  }
}
```

---

### B. Claude Code (CLI)

Adicione o servidor MCP via linha de comando:

```bash
claude mcp add credential-vault -- /caminho/para/credential-vault serve
```

Depois defina a variável de ambiente antes de iniciar o Claude Code:

```bash
export VAULT_PASSWORD="sua-senha-mestra"
claude
```

Ou adicione ao seu `~/.bashrc` / `~/.zshrc`:

```bash
export VAULT_PASSWORD="sua-senha-mestra"
```

---

### C. Claude Desktop

Edite `claude_desktop_config.json`:

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "credential-vault": {
      "command": "/caminho/para/credential-vault",
      "args": ["serve"],
      "env": {
        "VAULT_PASSWORD": "sua-senha-mestra"
      }
    }
  }
}
```

---

### D. Cursor

O Cursor suporta servidores MCP nas suas configurações.

**Via interface do Cursor**:
1. Abra Cursor → Settings → Features → MCP
2. Clique em "Add new MCP server"
3. Name: `credential-vault`
4. Type: `stdio`
5. Command: `/caminho/para/credential-vault`
6. Args: `serve`
7. Env: `VAULT_PASSWORD=sua-senha-mestra`

**Via arquivo de configuração** — edite `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "credential-vault": {
      "command": "/caminho/para/credential-vault",
      "args": ["serve"],
      "env": {
        "VAULT_PASSWORD": "sua-senha-mestra"
      }
    }
  }
}
```

---

### E. Windsurf (Codeium)

Edite `~/.codeium/windsurf/mcp_config.json`:

```json
{
  "mcpServers": {
    "credential-vault": {
      "command": "/caminho/para/credential-vault",
      "args": ["serve"],
      "env": {
        "VAULT_PASSWORD": "sua-senha-mestra"
      }
    }
  }
}
```

---

### F. Zed

Edite `~/.config/zed/settings.json` e adicione sob `context_servers`:

```json
{
  "context_servers": {
    "credential-vault": {
      "command": {
        "path": "/caminho/para/credential-vault",
        "args": ["serve"],
        "env": {
          "VAULT_PASSWORD": "sua-senha-mestra"
        }
      }
    }
  }
}
```

---

### G. Continue (VS Code / JetBrains)

Edite `~/.continue/config.json` e adicione sob `mcpServers`:

```json
{
  "mcpServers": {
    "credential-vault": {
      "command": "/caminho/para/credential-vault",
      "args": ["serve"],
      "env": {
        "VAULT_PASSWORD": "sua-senha-mestra"
      }
    }
  }
}
```

---

### H. Cline (VS Code)

O Cline usa o mesmo formato do Claude Desktop. Edite `~/Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json` (ou o caminho equivalente no seu sistema):

```json
{
  "mcpServers": {
    "credential-vault": {
      "command": "/caminho/para/credential-vault",
      "args": ["serve"],
      "env": {
        "VAULT_PASSWORD": "sua-senha-mestra"
      }
    }
  }
}
```

---

### I. Usando variável de ambiente do sistema (mais seguro)

Em vez de colocar a senha no arquivo de configuração, defina no perfil do shell (`~/.bashrc`, `~/.zshrc`):

```bash
export VAULT_PASSWORD="sua-senha-mestra"
```

Depois na configuração MCP, omita o bloco `environment` / `env`:

```json
{
  "mcp": {
    "credential-vault": {
      "type": "local",
      "command": ["/caminho/para/credential-vault", "serve"],
      "enabled": true
    }
  }
}
```

---

### Variáveis de Ambiente

| Variável | Obrigatória | Descrição |
|---|---|---|
| `VAULT_PASSWORD` | **Sim** (para `serve`) | Senha mestra para desencriptar o vault |
| `VAULT_PATH` | Não | Caminho do arquivo vault (padrão: `~/.config/credential-vault/vault.json`) |

---

## Passo 5: Usar via assistente de IA

Após configurar, reinicie seu assistente de IA. O LLM agora tem acesso às seguintes ferramentas:

### Ferramenta: `list_servers`

Lista os servidores cadastrados (nomes + descrições, sem credenciais).

**Exemplo de uso:**
```
Você: "Quais servidores estão disponíveis para deploy?"

IA: [chama list_servers]
    "Você tem 2 servidores cadastrados:
     - Server PROD 1: Servidor de produção principal
     - Server PROD 2: Servidor de produção secundário"
```

### Ferramenta: `get_connection_info`

Retorna host, porta e usuário de um servidor (sem senha/chave).

**Exemplo de uso:**
```
Você: "Qual o endereço do Server PROD 1?"

IA: [chama get_connection_info com server_name="Server PROD 1"]
    "Server PROD 1: root@192.168.1.10:22"
```

### Ferramenta: `deploy`

Conecta via SSH ao servidor e executa um comando de deploy.
As credenciais são lidas internamente do vault e **nunca** aparecem no contexto do LLM.

**Exemplo de uso:**
```
Você: "Faça deploy no Server PROD 1: execute docker compose pull && docker compose up -d"

IA: [chama deploy com server_name="Server PROD 1",
     command="docker compose pull && docker compose up -d"]

    "Deploy executado no Server PROD 1:
     stdout: Pulling web... done
             Starting web... done
     exit code: 0"
```

### Ferramenta: `ssh_exec`

Executa um comando arbitrário via SSH no servidor.

**Exemplo de uso:**
```
Você: "Verifique o status do nginx no Server PROD 1"

IA: [chama ssh_exec com server_name="Server PROD 1",
     command="systemctl status nginx"]

    "Comando executado no Server PROD 1:
     stdout: nginx.service - The nginx HTTP and reverse proxy server
              Active: active (running)
     exit code: 0"
```

---

## Ciclo de Vida do Servidor

O servidor MCP usa transporte **stdio** — **NÃO** abre porta de rede.

| Situação | O que acontece |
|---|---|
| Você abre o assistente de IA | credential-vault **inicia automaticamente** (como subprocesso) |
| Você fecha o assistente de IA | credential-vault **encerra automaticamente** |
| Não quer usar o vault | Só não chamar as ferramentas MCP — fica parado, sem consumo |
| Quer desativar temporariamente | Mudar `"enabled": false` na configuração |

Você **NÃO** precisa:
- Iniciar manualmente (`./credential-vault serve &`)
- Matar o processo manualmente
- Configurar serviço systemd

---

## Segurança

### O que a IA NUNCA vê

- Senhas
- Chaves privadas
- Conteúdo do arquivo do vault
- Senha mestra

### O que a IA vê

- Nomes e descrições dos servidores
- Host, porta e usuário (via `get_connection_info`)
- Resultado dos comandos (stdout/stderr)

### Como as credenciais são protegidas

1. **Em disco**: AES-256-GCM + Argon2id (criptografia autenticada)
2. **Em trânsito**: Senha mestra via variável de ambiente (não aparece no `ps`)
3. **Do LLM**: O vault conecta via SSH internamente — as credenciais nunca saem do processo

---

## Gerenciamento de Servidores

### Remover um servidor

```bash
./credential-vault delete "Server PROD 2"
```

### Atualizar um servidor

Use o comando `add` com o mesmo nome (sobrescreve):

```bash
./credential-vault add "Server PROD 1"
```

### Backup do vault

O vault é um único arquivo JSON criptografado. Para fazer backup:

```bash
cp ~/.config/credential-vault/vault.json /local/backup/vault-backup-$(date +%Y%m%d).json
```

> **Lembre-se**: Sem a senha mestra, o backup é inútil. Guarde a senha separadamente
> em um gerenciador de senhas.

---

## Troubleshooting

### Erro: "vault não encontrado"

```
Erro: vault não encontrado — execute 'credential-vault init' primeiro
```

**Solução**: Execute `./credential-vault init` para criar o vault.

### Erro: "VAULT_PASSWORD não definida"

```
Erro: VAULT_PASSWORD não definida — o servidor MCP requer a senha via variável de ambiente
```

**Solução**: Certifique-se de que `VAULT_PASSWORD` está definida no bloco `environment` da configuração MCP.

### Erro: "failed to decrypt (wrong password or corrupted data)"

A senha mestra está incorreta ou o arquivo do vault está corrompido.

**Solução**:
1. Verifique se a senha está correta
2. Se o vault estiver corrompido, restaure do backup com a senha correta

### O assistente de IA não vê as ferramentas

1. Verifique se o caminho do binário está correto na configuração
2. Verifique se o binário tem permissão de execução: `chmod +x credential-vault`
3. Reinicie o assistente de IA após alterar a configuração

---

## Resumo Rápido

```bash
# 1. Build
go build -o credential-vault ./cmd/credential-vault

# 2. Init vault
./credential-vault init

# 3. Add server
./credential-vault add "Server PROD 1"

# 4. Configure in opencode.json (see above)

# 5. Use via AI: "Deploy to Server PROD 1: run docker compose up -d"
```