# 🔗 NxtChain

NxtChain is a blockchain written in Go. It includes features such as block creation, transaction handling, and peer-to-peer synchronization.

## 🚀 Getting Started

### 📖 Prerequisites

-   Go 1.23.4 or later

### 📂 Installation

1. Clone the repository:

    ```sh
    git clone https://github.com/NXT-Crypto/nxtchain.git
    cd nxtchain
    ```

2. Install dependencies:
    ```sh
    go mod tidy
    ```

### 🔨 Building the Project

To build the project, simply run the provided buildscript `build.sh`:

```sh
bash ./build.sh
```

The buildscript will then build the NxtChain for multiple operating systems. You will find your built binaries in the `./build` folder.

If you are using Windows to build the NxtChain, use a tool such as [Git Bash](https://git-scm.com/downloads/win) or [WSL](https://learn.microsoft.com/en-us/windows/wsl/install) to run the buildscript. A batch file for Windows is **NOT** provided.

If you don't want to build the NxtChain yourself, you will always find a prepared build of the latest commits on the [releases page](https://github.com/NXT-Crypto/nxtchain/releases/tag/latest).
