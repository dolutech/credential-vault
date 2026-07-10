# Usage Guide

A practical, step-by-step guide to using **credential-vault** — from installation to daily usage.

> This guide is designed to get you productive quickly. For architecture details, see [CREDENTIAL_VAULT_PLAN.md](CREDENTIAL_VAULT_PLAN.md). For MCP configuration in specific AI assistants, see [MCP_CONFIG.md](MCP_CONFIG.md).

---

## Table of Contents

1. [Installation](#1-installation)
2. [Initialize the Vault](#2-initialize-the-vault)
3. [Add Servers](#3-add-servers)
4. [List Servers](#4-list-servers)
5. [Delete a Server](#5-delete-a-server)
6. [Start the MCP Server](#6-start-the-mcp-server)
7. [Configure in Your AI Assistant](#7-configure-in-your-ai-assistant)
8. [Daily Usage Examples](#8-daily-usage-examples)
9. [Vault File Management](#9-vault-file-management)
10. [Troubleshooting](#10-troubleshooting)

---

## 1. Installation

### Option A: Build from source (all platforms)

```bash
git clone https://github.com/dolutech/credential-vault.git
cd credential-vault
go build -o credential-vault ./cmd/credential-vault
```

### Option B: Install script (Linux & macOS)

```bash
git clone https://github.com/dolutech/credential-vault.git
cd credential-vault
./scripts/install.sh
```

This installs the binary to `/usr/local/bin` (if you have permissions) or `~/.local/bin`.

### Option C: Using Make

```bash
make build       # build for current platform
make build-all   # build for all 6 platforms (linux/darwin/windows × amd64/arm64)
make release     # create release archives in dist/
```

### Verify installation

```bash
./credential-vault --version
# credential-vault v0.1.0
```

---

## 2. Initialize the Vault

The vault is the encrypted file where your server credentials are stored.

```bash
./credential-vault init
```

You will be prompted to set a **master password** and confirm it:

```
Initializing vault at: /home/user/.config/credential-vault/vault.json
Master password: ********
Confirm master password: ********
Vault created successfully at: /home/user/.config/credential-vault/vault.json
Use the 'add' command to register servers.
```

### Important about the master password

- It is used to derive the AES-256 encryption key via Argon2id
- It is **NEVER** stored in any file
- Keep it in a password manager (1Password, Bitwarden, KeePass, etc.)
- If you lose it, the vault **cannot** be recovered

### Default vault file locations

| OS | Path |
|---|---|
| Linux | `~/.config/credential-vault/vault.json` (or `$XDG_CONFIG_HOME/...`) |
| macOS | `~/Library/Application Support/credential-vault/vault.json` |
| Windows | `%AppData%\credential-vault\vault.json` |

> To use a custom path, set the `VAULT_PATH` environment variable:
> ```bash
> VAULT_PATH=/my/custom/path/vault.json ./credential-vault init
> ```

---

## 3. Add Servers

Add each server using the `add` command in interactive mode:

```bash
./credential-vault add "Server PROD 1"
```

You will be prompted for each field:

```
Adding server: Server PROD 1

Master password: ********
Host/IP: 192.168.1.10
SSH port [22]: 22
User: root
Password (leave empty to use private key): my-secret-password
Description (optional): Main production server

Server 'Server PROD 1' added successfully!
```

### Using a private key instead of a password

Leave the "Password" field empty and paste the private key:

```bash
./credential-vault add "Server PROD 2"
```

```
Adding server: Server PROD 2

Master password: ********
Host/IP: 10.0.0.20
SSH port [22]: 2222
User: deploy
Password (leave empty to use private key):    ← press Enter (leave empty)
Paste private key content (Ctrl+D to finish):
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAA...
-----END OPENSSH PRIVATE KEY-----
Description (optional): Secondary production server

Server 'Server PROD 2' added successfully!
```

### Add as many servers as you need

```bash
./credential-vault add "Server PROD 1"
./credential-vault add "Server PROD 2"
./credential-vault add "Server STAGING 1"
./credential-vault add "Server DEV 1"
./credential-vault add "Server HOMOLOG 1"
```

### Update a server

Use the `add` command with the same name — it overwrites the previous entry:

```bash
./credential-vault add "Server PROD 1"
# Fill in the new credentials
```

---

## 4. List Servers

View all registered servers:

```bash
./credential-vault list
```

Output:

```
Registered servers (5):

  - Server PROD 1: Main production server
  - Server PROD 2: Secondary production server
  - Server STAGING 1: Staging environment
  - Server DEV 1: Development server
  - Server HOMOLOG 1: Homologation server
```

> **The `list` command NEVER displays credentials** — only names and descriptions.

### Tip: set VAULT_PASSWORD to avoid typing it every time

```bash
export VAULT_PASSWORD="your-master-password"
./credential-vault list       # no password prompt
./credential-vault add "..."  # no password prompt
```

---

## 5. Delete a Server

Remove a server from the vault:

```bash
./credential-vault delete "Server DEV 1"
```

Output:

```
Server 'Server DEV 1' removed successfully.
```

If the server doesn't exist:

```
Error: server 'Server INEXISTENT' not found in vault
```

---

## 6. Start the MCP Server

The `serve` command starts the MCP server on stdio, ready to communicate with your AI assistant.

```bash
VAULT_PASSWORD="your-master-password" ./credential-vault serve
```

> **You normally don't run this manually** — your AI assistant (opencode, Claude, Cursor, etc.) starts it automatically as a subprocess. See the next step.

The `serve` command requires `VAULT_PASSWORD` as an environment variable (not as a CLI argument, so it doesn't appear in `ps`).

---

## 7. Configure in Your AI Assistant

Add the credential-vault as an MCP server in your AI assistant's config.

### opencode (global)

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

### Claude Code

```bash
claude mcp add credential-vault -- /path/to/credential-vault serve
export VAULT_PASSWORD="your-master-password"
```

### Claude Desktop, Cursor, Windsurf, Zed, Continue, Cline

See the full configuration guide: **[MCP_CONFIG.md](MCP_CONFIG.md)** — covers all 8 supported AI assistants with exact config examples.

> **Lifecycle**: The vault starts automatically when you open your AI assistant, and stops when you close it. No background process, no network port, no manual start/stop needed.

---

## 8. Daily Usage Examples

Once configured, just talk to your AI assistant naturally:

### List available servers

```
You: "Which servers can I deploy to?"

AI: [calls list_servers]
    "You have 3 servers registered:
     - Server PROD 1: Main production server
     - Server STAGING 1: Staging environment
     - Server DEV 1: Development server"
```

### Get connection info (no password)

```
You: "What's the connection info for Server PROD 1?"

AI: [calls get_connection_info with server_name="Server PROD 1"]
    "Server PROD 1: root@192.168.1.10:22 — Main production server"
```

### Deploy to a server

```
You: "Deploy to Server PROD 1: run docker compose pull && docker compose up -d"

AI: [calls deploy with server_name="Server PROD 1",
     command="docker compose pull && docker compose up -d"]

    "Deploy executed on Server PROD 1:
     stdout: Pulling web... done
             Starting web... done
     exit code: 0"
```

### Run a diagnostic command

```
You: "Check nginx status on Server PROD 1"

AI: [calls ssh_exec with server_name="Server PROD 1",
     command="systemctl status nginx"]

    "Command executed on Server PROD 1:
     stdout: nginx.service - The nginx HTTP and reverse proxy server
              Active: active (running)
     exit code: 0"
```

### Check disk space on all servers

```
You: "Check disk space on Server PROD 1"

AI: [calls ssh_exec with server_name="Server PROD 1",
     command="df -h"]

    "Command executed on Server PROD 1:
     stdout: Filesystem      Size  Used Avail Use% Mounted on
             /dev/sda1        50G   30G   20G  60% /
     exit code: 0"
```

### Restart a service

```
You: "Restart the web service on Server STAGING 1"

AI: [calls ssh_exec with server_name="Server STAGING 1",
     command="sudo systemctl restart nginx"]

    "Command executed on Server STAGING 1:
     stdout: (empty)
     stderr: (empty)
     exit code: 0"
```

### Multi-step deploy

```
You: "Deploy to Server PROD 1: 
      1. git pull origin main
      2. npm install
      3. npm run build
      4. pm2 restart all"

AI: [calls deploy with server_name="Server PROD 1",
     command="git pull origin main && npm install && npm run build && pm2 restart all"]

    "Deploy executed on Server PROD 1:
     stdout: Updating origin/main...
             added 5 packages in 3s
             Build complete
             [PM2] Restarting all processes
     exit code: 0"
```

---

## 9. Vault File Management

### Backup the vault

The vault is a single encrypted JSON file. Back it up anytime:

```bash
# Linux
cp ~/.config/credential-vault/vault.json ~/backups/vault-backup-$(date +%Y%m%d).json

# macOS
cp ~/Library/Application\ Support/credential-vault/vault.json ~/backups/vault-backup-$(date +%Y%m%d).json

# Windows (PowerShell)
Copy-Item "$env:APPDATA\credential-vault\vault.json" "$HOME\backups\vault-backup-$(Get-Date -Format yyyyMMdd).json"
```

> **Remember**: Without the master password, the backup is useless. Store the password separately in a password manager.

### Restore the vault

```bash
# Stop your AI assistant first (so the MCP server is not running)
# Then restore:
cp ~/backups/vault-backup-20260710.json ~/.config/credential-vault/vault.json
```

### Move the vault to a different machine

1. Copy the `vault.json` file to the new machine
2. Ensure `VAULT_PASSWORD` is set on the new machine
3. Set `VAULT_PATH` to the new location if different

```bash
# On the new machine
VAULT_PATH=/path/to/copied/vault.json VAULT_PASSWORD=your-password ./credential-vault list
```

### Verify the vault is encrypted

```bash
cat ~/.config/credential-vault/vault.json
```

You should see only the encrypted structure — no readable passwords:

```json
{
  "version": 1,
  "kdf": {
    "algorithm": "argon2id",
    "salt": "U0DZocl9p0zkQNshSr+UeA==",
    "memory": 65536,
    "iterations": 3,
    "parallelism": 2
  },
  "ciphertext": "hEWp4sLOrPah37gk5NVrImvgtHjnmHU0Hnvr5tMn55/..."
}
```

---

## 10. Troubleshooting

### "vault not found — run 'credential-vault init' first"

The vault file doesn't exist. Run:
```bash
./credential-vault init
```

### "VAULT_PASSWORD not set"

The `serve` command requires the password as an environment variable:
```bash
VAULT_PASSWORD="your-password" ./credential-vault serve
```

Or set it permanently in your shell profile (`~/.bashrc`, `~/.zshrc`):
```bash
export VAULT_PASSWORD="your-password"
```

### "failed to decrypt (wrong password or corrupted data)"

The master password is incorrect, or the vault file is corrupted.

1. Check that the password is correct
2. If corrupted, restore from backup:
   ```bash
   cp ~/backups/vault-backup-YYYYMMDD.json ~/.config/credential-vault/vault.json
   ```

### "cannot divide by zero" / SSH connection failed

The server credentials might be wrong or the server is unreachable:
1. Check that the host and port are correct: `./credential-vault list`
2. Test the connection manually:
   ```bash
   ssh -p 22 root@192.168.1.10
   ```
3. If using a private key, ensure it was pasted correctly

### AI assistant doesn't see the tools

1. Verify the binary path is correct in your MCP config
2. Verify the binary has execute permission: `chmod +x credential-vault`
3. Restart your AI assistant after changing the config
4. Check that `VAULT_PASSWORD` is set in the MCP config's `environment` block

### SSH host key verification failed

If the server's host key is not in `~/.ssh/known_hosts`, connect manually first:
```bash
ssh -p 22 root@192.168.1.10
# Type "yes" to accept the host key
```

Then the vault will be able to verify it automatically.

---

## Quick Reference Card

| Action | Command |
|---|---|
| Install | `go build -o credential-vault ./cmd/credential-vault` |
| Init vault | `./credential-vault init` |
| Add server | `./credential-vault add "Server PROD 1"` |
| List servers | `./credential-vault list` |
| Delete server | `./credential-vault delete "Server PROD 1"` |
| Start MCP | `VAULT_PASSWORD=xxx ./credential-vault serve` |
| Show version | `./credential-vault --version` |
| Show help | `./credential-vault help` |

---

<br>
<br>
<br>

---

# Guia de Utilização (Português)

Um guia prático e passo-a-passo para usar o **credential-vault** — desde a instalação até o uso diário.

> Este guia foi desenhado para te deixar produtivo rapidamente. Para detalhes de arquitetura, veja [CREDENTIAL_VAULT_PLAN.md](CREDENTIAL_VAULT_PLAN.md). Para configuração MCP em assistentes de IA específicos, veja [MCP_CONFIG.md](MCP_CONFIG.md).

---

## Índice

1. [Instalação](#1-instalação-1)
2. [Inicializar o Cofre](#2-inicializar-o-cofre-1)
3. [Adicionar Servidores](#3-adicionar-servidores-1)
4. [Listar Servidores](#4-listar-servidores-1)
5. [Remover um Servidor](#5-remover-um-servidor-1)
6. [Iniciar o Servidor MCP](#6-iniciar-o-servidor-mcp-1)
7. [Configurar no seu Assistente de IA](#7-configurar-no-seu-assistente-de-ia-1)
8. [Exemplos de Uso Diário](#8-exemplos-de-uso-diário-1)
9. [Gerenciamento do Arquivo do Cofre](#9-gerenciamento-do-arquivo-do-cofre-1)
10. [Resolução de Problemas](#10-resolução-de-problemas-1)

---

## 1. Instalação

### Opção A: Compilar do código-fonte (todas as plataformas)

```bash
git clone https://github.com/dolutech/credential-vault.git
cd credential-vault
go build -o credential-vault ./cmd/credential-vault
```

### Opção B: Script de instalação (Linux & macOS)

```bash
git clone https://github.com/dolutech/credential-vault.git
cd credential-vault
./scripts/install.sh
```

Instala o binário em `/usr/local/bin` (se tiver permissões) ou `~/.local/bin`.

### Opção C: Usando Make

```bash
make build       # compilar para a plataforma atual
make build-all   # compilar para as 6 plataformas (linux/darwin/windows × amd64/arm64)
make release     # criar pacotes de release em dist/
```

### Verificar a instalação

```bash
./credential-vault --version
# credential-vault v0.1.0
```

---

## 2. Inicializar o Cofre

O cofre é o arquivo encriptado onde suas credenciais de servidores ficam armazenadas.

```bash
./credential-vault init
```

Você será solicitado a definir uma **senha mestra** e confirmá-la:

```
Initializing vault at: /home/user/.config/credential-vault/vault.json
Master password: ********
Confirm master password: ********
Vault created successfully at: /home/user/.config/credential-vault/vault.json
Use the 'add' command to register servers.
```

### Importante sobre a senha mestra

- É usada para derivar a chave de criptografia AES-256 via Argon2id
- **NUNCA** é armazenada em nenhum arquivo
- Guarde-a num gerenciador de senhas (1Password, Bitwarden, KeePass, etc.)
- Se a perder, o cofre **não pode** ser recuperado

### Localizações padrão do arquivo do cofre

| SO | Caminho |
|---|---|
| Linux | `~/.config/credential-vault/vault.json` (ou `$XDG_CONFIG_HOME/...`) |
| macOS | `~/Library/Application Support/credential-vault/vault.json` |
| Windows | `%AppData%\credential-vault\vault.json` |

> Para usar um caminho personalizado, defina a variável `VAULT_PATH`:
> ```bash
> VAULT_PATH=/meu/caminho/personalizado/vault.json ./credential-vault init
> ```

---

## 3. Adicionar Servidores

Adicione cada servidor usando o comando `add` em modo interativo:

```bash
./credential-vault add "Server PROD 1"
```

Você será solicitado a preencher cada campo:

```
Adding server: Server PROD 1

Master password: ********
Host/IP: 192.168.1.10
SSH port [22]: 22
User: root
Password (leave empty to use private key): minha-senha-secreta
Description (optional): Servidor de produção principal

Server 'Server PROD 1' added successfully!
```

### Usando chave privada em vez de senha

Deixe o campo "Password" vazio e cole a chave privada:

```bash
./credential-vault add "Server PROD 2"
```

```
Adding server: Server PROD 2

Master password: ********
Host/IP: 10.0.0.20
SSH port [22]: 2222
User: deploy
Password (leave empty to use private key):    ← dê Enter (deixe vazio)
Paste private key content (Ctrl+D to finish):
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAA...
-----END OPENSSH PRIVATE KEY-----
Description (optional): Servidor de produção secundário

Server 'Server PROD 2' added successfully!
```

### Adicione quantos servidores precisar

```bash
./credential-vault add "Server PROD 1"
./credential-vault add "Server PROD 2"
./credential-vault add "Server STAGING 1"
./credential-vault add "Server DEV 1"
./credential-vault add "Server HOMOLOG 1"
```

### Atualizar um servidor

Use o comando `add` com o mesmo nome — ele sobrescreve a entrada anterior:

```bash
./credential-vault add "Server PROD 1"
# Preenche as novas credenciais
```

---

## 4. Listar Servidores

Veja todos os servidores cadastrados:

```bash
./credential-vault list
```

Saída:

```
Registered servers (5):

  - Server PROD 1: Servidor de produção principal
  - Server PROD 2: Servidor de produção secundário
  - Server STAGING 1: Ambiente de homologação
  - Server DEV 1: Servidor de desenvolvimento
  - Server HOMOLOG 1: Servidor de homologação
```

> **O comando `list` NUNCA exibe credenciais** — apenas nomes e descrições.

### Dica: defina VAULT_PASSWORD para não digitar a senha toda vez

```bash
export VAULT_PASSWORD="sua-senha-mestra"
./credential-vault list       # sem pedir senha
./credential-vault add "..."  # sem pedir senha
```

---

## 5. Remover um Servidor

Remova um servidor do cofre:

```bash
./credential-vault delete "Server DEV 1"
```

Saída:

```
Server 'Server DEV 1' removed successfully.
```

Se o servidor não existir:

```
Error: server 'Server INEXISTENTE' not found in vault
```

---

## 6. Iniciar o Servidor MCP

O comando `serve` inicia o servidor MCP em stdio, pronto para comunicar com seu assistente de IA.

```bash
VAULT_PASSWORD="sua-senha-mestra" ./credential-vault serve
```

> **Normalmente você não roda isso manualmente** — seu assistente de IA (opencode, Claude, Cursor, etc.) inicia-o automaticamente como subprocesso. Veja o próximo passo.

O comando `serve` requer `VAULT_PASSWORD` como variável de ambiente (não como argumento de CLI, para não aparecer no `ps`).

---

## 7. Configurar no seu Assistente de IA

Adicione o credential-vault como servidor MCP na configuração do seu assistente de IA.

### opencode (global)

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

### Claude Code

```bash
claude mcp add credential-vault -- /caminho/para/credential-vault serve
export VAULT_PASSWORD="sua-senha-mestra"
```

### Claude Desktop, Cursor, Windsurf, Zed, Continue, Cline

Veja o guia completo de configuração: **[MCP_CONFIG.md](MCP_CONFIG.md)** — cobre os 8 assistentes de IA suportados com exemplos exatos.

> **Ciclo de vida**: O cofre inicia automaticamente quando você abre o assistente de IA, e encerra quando você fecha. Sem processo em background, sem porta de rede, sem start/stop manual.

---

## 8. Exemplos de Uso Diário

Depois de configurado, basta falar com seu assistente de IA naturalmente:

### Listar servidores disponíveis

```
Você: "Quais servidores posso fazer deploy?"

IA: [chama list_servers]
    "Você tem 3 servidores cadastrados:
     - Server PROD 1: Servidor de produção principal
     - Server STAGING 1: Ambiente de homologação
     - Server DEV 1: Servidor de desenvolvimento"
```

### Obter info de conexão (sem senha)

```
Você: "Qual a conexão do Server PROD 1?"

IA: [chama get_connection_info com server_name="Server PROD 1"]
    "Server PROD 1: root@192.168.1.10:22 — Servidor de produção principal"
```

### Fazer deploy num servidor

```
Você: "Faça deploy no Server PROD 1: execute docker compose pull && docker compose up -d"

IA: [chama deploy com server_name="Server PROD 1",
     command="docker compose pull && docker compose up -d"]

    "Deploy executado no Server PROD 1:
     stdout: Pulling web... done
             Starting web... done
     exit code: 0"
```

### Rodar um comando de diagnóstico

```
Você: "Verifique o status do nginx no Server PROD 1"

IA: [chama ssh_exec com server_name="Server PROD 1",
     command="systemctl status nginx"]

    "Comando executado no Server PROD 1:
     stdout: nginx.service - The nginx HTTP and reverse proxy server
              Active: active (running)
     exit code: 0"
```

### Verificar espaço em disco

```
Você: "Verifique o espaço em disco no Server PROD 1"

IA: [chama ssh_exec com server_name="Server PROD 1",
     command="df -h"]

    "Comando executado no Server PROD 1:
     stdout: Filesystem      Size  Used Avail Use% Mounted on
             /dev/sda1        50G   30G   20G  60% /
     exit code: 0"
```

### Reiniciar um serviço

```
Você: "Reinicia o serviço web no Server STAGING 1"

IA: [chama ssh_exec com server_name="Server STAGING 1",
     command="sudo systemctl restart nginx"]

    "Comando executado no Server STAGING 1:
     stdout: (vazio)
     stderr: (vazio)
     exit code: 0"
```

### Deploy multi-etapa

```
Você: "Faça deploy no Server PROD 1:
      1. git pull origin main
      2. npm install
      3. npm run build
      4. pm2 restart all"

IA: [chama deploy com server_name="Server PROD 1",
     command="git pull origin main && npm install && npm run build && pm2 restart all"]

    "Deploy executado no Server PROD 1:
     stdout: Updating origin/main...
             added 5 packages in 3s
             Build complete
             [PM2] Restarting all processes
     exit code: 0"
```

---

## 9. Gerenciamento do Arquivo do Cofre

### Backup do cofre

O cofre é um único arquivo JSON encriptado. Faça backup quando quiser:

```bash
# Linux
cp ~/.config/credential-vault/vault.json ~/backups/vault-backup-$(date +%Y%m%d).json

# macOS
cp ~/Library/Application\ Support/credential-vault/vault.json ~/backups/vault-backup-$(date +%Y%m%d).json

# Windows (PowerShell)
Copy-Item "$env:APPDATA\credential-vault\vault.json" "$HOME\backups\vault-backup-$(Get-Date -Format yyyyMMdd).json"
```

> **Lembre-se**: Sem a senha mestra, o backup é inútil. Guarde a senha separadamente num gerenciador de senhas.

### Restaurar o cofre

```bash
# Pare seu assistente de IA primeiro (para que o servidor MCP não esteja rodando)
# Depois restaure:
cp ~/backups/vault-backup-20260710.json ~/.config/credential-vault/vault.json
```

### Mover o cofre para outra máquina

1. Copie o arquivo `vault.json` para a nova máquina
2. Garanta que `VAULT_PASSWORD` está definido na nova máquina
3. Defina `VAULT_PATH` para o novo local se for diferente

```bash
# Na nova máquina
VAULT_PATH=/caminho/do/vault/copiado.json VAULT_PASSWORD=sua-senha ./credential-vault list
```

### Verificar que o cofre está encriptado

```bash
cat ~/.config/credential-vault/vault.json
```

Você deve ver apenas a estrutura encriptada — sem senhas legíveis:

```json
{
  "version": 1,
  "kdf": {
    "algorithm": "argon2id",
    "salt": "U0DZocl9p0zkQNshSr+UeA==",
    "memory": 65536,
    "iterations": 3,
    "parallelism": 2
  },
  "ciphertext": "hEWp4sLOrPah37gk5NVrImvgtHjnmHU0Hnvr5tMn55/..."
}
```

---

## 10. Resolução de Problemas

### "vault not found — run 'credential-vault init' first"

O arquivo do cofre não existe. Execute:
```bash
./credential-vault init
```

### "VAULT_PASSWORD not set"

O comando `serve` requer a senha como variável de ambiente:
```bash
VAULT_PASSWORD="sua-senha" ./credential-vault serve
```

Ou defina permanentemente no seu perfil do shell (`~/.bashrc`, `~/.zshrc`):
```bash
export VAULT_PASSWORD="sua-senha"
```

### "failed to decrypt (wrong password or corrupted data)"

A senha mestra está incorreta, ou o arquivo do cofre está corrompido.

1. Verifique se a senha está correta
2. Se corrompido, restaure do backup:
   ```bash
   cp ~/backups/vault-backup-YYYYMMDD.json ~/.config/credential-vault/vault.json
   ```

### Falha na conexão SSH

As credenciais do servidor podem estar erradas ou o servidor inacessível:
1. Verifique que o host e a porta estão corretos: `./credential-vault list`
2. Teste a conexão manualmente:
   ```bash
   ssh -p 22 root@192.168.1.10
   ```
3. Se usar chave privada, garanta que foi colada corretamente

### O assistente de IA não vê as ferramentas

1. Verifique se o caminho do binário está correto na configuração MCP
2. Verifique se o binário tem permissão de execução: `chmod +x credential-vault`
3. Reinicie o assistente de IA após alterar a configuração
4. Confirme que `VAULT_PASSWORD` está definido no bloco `environment` da configuração MCP

### Falha na verificação de chave de host SSH

Se a chave de host do servidor não estiver em `~/.ssh/known_hosts`, conecte manualmente primeiro:
```bash
ssh -p 22 root@192.168.1.10
# Digite "yes" para aceitar a chave de host
```

Depois o cofre conseguirá verificá-la automaticamente.

---

## Cartão de Referência Rápida

| Ação | Comando |
|---|---|
| Instalar | `go build -o credential-vault ./cmd/credential-vault` |
| Iniciar cofre | `./credential-vault init` |
| Adicionar servidor | `./credential-vault add "Server PROD 1"` |
| Listar servidores | `./credential-vault list` |
| Remover servidor | `./credential-vault delete "Server PROD 1"` |
| Iniciar MCP | `VAULT_PASSWORD=xxx ./credential-vault serve` |
| Mostrar versão | `./credential-vault --version` |
| Mostrar ajuda | `./credential-vault help` |