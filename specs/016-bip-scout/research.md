# Research: bip scout

## R1: Native SSH with ProxyJump in Go

**Decision**: Use `golang.org/x/crypto/ssh` with manual ProxyJump.

ProxyJump is implemented by dialing the jump host first, then tunneling the target connection through it:

```go
jumpClient, _ := ssh.Dial("tcp", "jump:22", jumpConfig)
targetConn, _ := jumpClient.Dial("tcp", "target:22")
targetClient, _ := ssh.NewClientConn(targetConn, "target:22", targetConfig)
client := ssh.NewClient(targetClient, chans, reqs)
```

Auth via SSH agent (`SSH_AUTH_SOCK` → `golang.org/x/crypto/ssh/agent`). No key file paths in config.

**Rejected**: subprocess `ssh` (FR-008 mandates native), `github.com/kevinburke/ssh_config` (unnecessary dep).

## R2: YAML Parsing

**Decision**: `gopkg.in/yaml.v3` — standard Go YAML library. Project currently uses JSON-only but spec mandates `servers.yml`.

**Rejected**: JSON config (conflicts with spec), `github.com/goccy/go-yaml` (non-standard for tiny config).

## R3: Remote Command Strategy

**Decision**: Hardcode commands in Go. No YAML-configurable commands.

Commands combined via `___SCOUT_DELIM___` separator in single SSH session:
- CPU: `top -bn1 | grep -i "cpu(s)" | awk '{print $2}' | cut -d'%' -f1`
- Memory: `free -m | awk '/^Mem:/ {printf "%.1f", ($3/$2) * 100}'`
- Load: `uptime | awk -F'load average:' '{print $2}' | sed 's/^[[:space:]]*//'`
- GPU util: `nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits`
- GPU mem: `nvidia-smi --query-gpu=memory.used,memory.total --format=csv,noheader,nounits`

**Rejected**: Configurable commands (issue #70 says hardcode).

## R4: Pattern Expansion

**Decision**: Regex `(.+)\{(\d+)\.\.(\d+)\}` → zero-padded iteration.

`beetle{01..05}` → beetle01, beetle02, beetle03, beetle04, beetle05. Padding determined by start value width.

## R5: SSH Error Reporting

**Decision**: Detect error types and produce actionable messages (FR-015):
- No agent → tell user to start `ssh-agent`
- No keys → tell user to run `ssh-add`
- Auth rejected → check `~/.ssh/config`
- Proxy timeout → name the proxy in error
- Server timeout → mark "offline", continue
