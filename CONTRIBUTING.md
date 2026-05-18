# Contributing to QBFile

Thank you for your interest in contributing to QBFile! We welcome contributions from developers, security auditors, designers, and documentation writers.

By contributing to this project, you agree to abide by our Code of Conduct and license terms.

## How Can I Contribute?

### 1. Reporting Bugs
* Search the existing issues list to make sure the bug hasn't already been reported.
* If it's a new bug, open a new issue using our **Bug Report** template.
* Provide a clear description, reproduction steps, and system details.

### 2. Suggesting Features
* Open an issue using our **Feature Request** template.
* Explain the use case, why this feature is beneficial, and how it should work.

### 3. Submitting Pull Requests (Code Contributions)
We follow a standard GitHub Fork-and-Pull model:

1. **Fork** the repository on GitHub.
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/qbfile.git
   ```
3. Create a descriptive **feature branch** for your edits:
   ```bash
   git checkout -b feature/your-awesome-feature
   ```
4. **Develop and Test** your changes:
   * Keep Go code formatted with `go fmt`.
   * Keep frontend code clean and responsive.
   * Run Wails dev mode to test bindings: `wails dev`.
5. **Commit** your changes using descriptive commit messages:
   ```bash
   git commit -m "feat: Add custom background presets to chat window"
   ```
6. **Push** to your feature branch:
   ```bash
   git push origin feature/your-awesome-feature
   ```
7. Open a **Pull Request (PR)** against our `main` branch.

## Development Stack
* **Backend**: Go (Golang) v1.21+
* **Frontend**: Svelte (v4), Vite, HTML, CSS (Vanilla)
* **P2P Transport**: Raw TCP and UDP sockets
* **Desktop Wrapper**: Wails framework (v2)
