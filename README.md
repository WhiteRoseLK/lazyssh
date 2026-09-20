<div align="center">
  <h1>🚀 neossh</h1>
  <p><b>A modern terminal-based SSH manager inspired by lazydocker and k9s</b></p>
  <p><i>The actively maintained successor to lazyssh</i></p>
</div>

<div align="center">

[![GitHub release](https://img.shields.io/github/v/release/WhiteRoseLK/neossh?style=flat-square)](https://github.com/WhiteRoseLK/neossh/releases)
[![License](https://img.shields.io/github/license/WhiteRoseLK/neossh?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/WhiteRoseLK/neossh?style=flat-square)](go.mod)

</div>

---

## 💡 About neossh

**neossh** is an interactive, keyboard-driven SSH manager for your terminal. With neossh, you can quickly navigate, connect, manage, and configure servers defined in your `~/.ssh/config` without remembering IP addresses or dealing with complex SSH commands.

### The lazyssh Lineage (Neovim & Vim style)

`neossh` is the independent, actively maintained successor to [Adembc/lazyssh](https://github.com/Adembc/lazyssh).

When the original lazyssh project became inactive with a backlog of unmerged community contributions, `neossh` was created to carry the torch:
- 🔄 **Community PRs integrated**: Full SSH `Include` support, fuzzy search, XDG directory compliance, custom config flags, and critical UI/navigation bug fixes.
- ⚡ **Zero friction migration**: `neossh` automatically detects and migrates your existing favorites, connection history, and tags from `~/.lazyssh` to `~/.neossh`.
- 🛠️ **Active maintenance**: Regular releases, responsive issue triage, and continuous improvements.

---

## 📷 Screenshots

<details>
<summary>Click to expand screenshots</summary>

### Startup
<img src="./docs/loader.png" alt="Startup screen" />

### Server List
<img src="./docs/list server.png" alt="Server list" />

### Fuzzy Search
<img src="./docs/search.png" alt="Fuzzy search" />

### SSH Connection
<img src="./docs/ssh.png" alt="SSH connection" />

### Add Server
<img src="./docs/add server.png" alt="Add server form" />

</details>

---

## ✨ Features

### Server Management
- 📜 Read & display servers from your `~/.ssh/config` in a clean, scrollable list.
- ➕ Add a new server from the UI with comprehensive SSH configuration options.
- ✏ Edit existing server entries directly from the UI with a tabbed interface.
- 🗑 Delete server entries safely.
- 📌 Pin / unpin servers to keep favorites at the top.
- 🏓 Ping server to check reachability.

### Quick Server Navigation
- 🔍 Fuzzy search by alias, IP, or tags (`/`).
- 🖥 One‑keypress SSH into the selected server (`Enter`).
- 🏷 Tag servers (e.g., `prod`, `dev`, `test`) for quick filtering.
- ↕️ Sort by alias or last SSH (toggle + reverse).
- 🔢 Jump focus between panels using numeric shortcuts (`1`, `2`, `3`).

### Advanced SSH Configuration
- 🔗 Port forwarding (`LocalForward`, `RemoteForward`, `DynamicForward`).
- 🚀 Connection multiplexing for instant subsequent connections.
- 🔐 Advanced authentication options (public key, password, agent forwarding).
- 🔒 Security settings (ciphers, MACs, key exchange algorithms).
- 🌐 Proxy settings (`ProxyJump`, `ProxyCommand`).
- ⚙️ Full SSH config options organized in a tabbed interface.

### Key Management
- 🔑 SSH key autocomplete with automatic detection of available keys in `~/.ssh/`.
- 📝 Smart key selection with support for multiple identity files.

### Enhancements over lazyssh
- 📂 **Full `Include` directive support**: Hosts defined in included files (e.g. `~/.ssh/config.d/*`) appear seamlessly, and writes safely route back to the original source file.
- 📋 **Copy SSH command**: Copy the full SSH command to clipboard with a single key (`c`).
- 💾 **Persistent sort mode**: Your preferred sort order is remembered across sessions.
- 🗂 **Custom SSH config**: `--sshconfig <path>` flag to target any configuration file.
- 📦 **XDG Base Directory compliant**: Respects `$XDG_CONFIG_HOME` and `$XDG_STATE_HOME`.

---

## 🚀 Installation

### Homebrew (macOS & Linux)

```bash
brew install WhiteRoseLK/tap/neossh
```

*(If you previously had `lazyssh` installed, Homebrew will seamlessly prompt to replace it while preserving your server configs).*

### Go Install

```bash
go install github.com/WhiteRoseLK/neossh/cmd@latest
```

### Pre-compiled Binaries

Download ready-to-run binaries for macOS, Linux, and Windows on the [Releases page](https://github.com/WhiteRoseLK/neossh/releases).

---

## ⌨️ Keybindings

| Key | Action |
|:---:|--------|
| `Enter` | SSH into selected server |
| `/` | Fuzzy search by alias, IP, or tag |
| `a` | Add new server |
| `e` | Edit selected server |
| `d` | Delete selected server |
| `p` | Pin / unpin server |
| `t` | Edit tags |
| `c` | Copy SSH connection command to clipboard |
| `s` | Toggle sort mode (alias, last SSH, reverse) |
| `P` | Ping selected server |
| `1` / `2` / `3` | Focus Search / Server List / Details |
| `j` / `k` or `↓` / `↑` | Navigate server list |
| `q` / `Ctrl+C` | Quit |

---

## 🔐 Security Notice

`neossh` does not introduce any new security risks. It is a TUI wrapper around your existing `~/.ssh/config` file:

- All SSH connections are executed through your system's native `ssh` binary (OpenSSH).
- Private keys, passwords, and credentials are never stored, transmitted, or inspected by `neossh`.
- Your existing `IdentityFile` paths and `ssh-agent` integrations work exactly as before.
- File permissions on your SSH config (`0600`) are strictly preserved.

---

## 🛡️ Config Safety: Non‑destructive writes and backups

- **Non‑destructive edits**: `neossh` only writes the minimal required changes to your `~/.ssh/config`. Comments, spacing, indentation, and untouched settings remain intact.
- **Atomic writes**: Updates are written to a temporary file and atomically renamed over the original to prevent corruption.
- **Backups**:
  - *One‑time snapshot*: Before `neossh` makes its first change, it creates a snapshot named `config.original.backup`. This file is never overwritten.
  - *Rolling backups*: On each save, `neossh` creates a timestamped backup (`~/.ssh/config-<timestamp>-neossh.backup`), keeping the 10 most recent.

---

## 📂 SSH Config `Include` Support

`neossh` honours top-level `Include` directives in your `~/.ssh/config`:

- **Reads**: All included files are parsed in OpenSSH precedence order.
- **Writes route back to source**: Editing or deleting a host modifies the file that actually defines it. Other files are never touched.
- **Ambiguity modal**: If the same alias is defined in multiple files, a prompt asks which file to update. Your choice is remembered in `metadata.json`.

---

## 🤝 Contributing

Contributions are welcome! Feel free to open an [Issue](https://github.com/WhiteRoseLK/neossh/issues) or submit a Pull Request.

---

## 📄 License

This project is licensed under the [Apache-2.0 License](LICENSE).

### Acknowledgments

- Originally created by [Adembc](https://github.com/Adembc) as [lazyssh](https://github.com/Adembc/lazyssh).
- Huge thanks to all upstream community contributors whose PRs and feedback made `neossh` possible.
