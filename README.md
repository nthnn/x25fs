# x25fs: Encrypted and Secure FUSE Filesystem

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

x25fs is a FUSE (Filesystem in Userspace) implementation that provides an encrypted and secure file storage solution. It leverages the `github.com/nthnn/xbin25` for encryption and decryption. Data is persisted in a single disk image file, offering a convenient way to manage encrypted storage.

* **Encryption:** Uses configurable encryption (via `github.com/nthnn/xbin25`) to secure all data stored in the filesystem. Supports encryption contexts/labels.
* **Data Integrity:** Relies on the underlying encryption library for data integrity.
* **POSIX-like File Operations:** Supports standard file system operations such as:
    * Creating, reading, writing, and deleting files.
    * Creating and removing directories.
    * Setting and getting file/directory attributes (permissions, timestamps, etc.).
    * Renaming files and directories.
* **Mountpoint Security:** Performs security checks on the mount point to prevent potential vulnerabilities (e.g., mounting on symlinks or world-writable directories).
* **Atomic Writes:** Uses `github.com/google/renameio` to ensure atomic writes of the disk image, preventing data corruption.
* **Disk Image Persistence:** Stores the entire filesystem in a single file for easy backup and management.
* **Inode Management:** Dynamically allocates inodes.
* **File Size Limits:** Enforces a configurable maximum file size to prevent excessive disk usage.

## Installation

### Building from Source

1.  **Install Go:** Ensure you have Go (version 1.16 or later) installed.

2.  **Clone the repository:**
    ```bash
    git clone [https://github.com/nthnn/x25fs.git](https://github.com/nthnn/x25fs.git)
    cd x25fs
    ```

3.  **Build the binary:**
    ```bash
    go build -o x25fs
    ```

4.  **Install dependencies:** (Usually go mod will handle this automatically on build)
    ```bash
    go mod tidy
    ```

### Downloading Build

1. **Download x25fs:** Go to [release](https://github.com/nthnn/x25fs/releases) page to download x25fs.

2. **Install using `dpkg`:** Install x25fs using `dpkg` as shown below:
    ```bash
    sudo dpkg -i x25fs-1.0.0.deb
    ```

## Usage

```
Usage: x25fs [options] <mountpoint>
```

### Options

* `--encrypt-cert <PEM file>`:  PEM file for encryption public key.
* `--encrypt-key <PEM file>`:  PEM file for decryption private key.
* `--sign-cert <PEM file>`:   PEM file for signature public key (optional).
* `--sign-key <PEM file>`:    PEM file for signing private key (optional).
* `--label <string>`:       Context label for OAEP encryption (optional).
* `--duration <duration>`:  Max age for replay protection (default: 36h).
* `--block-size <int>`:     Compression block size (default: 1024\*1024).  *Note: Compression is not currently implemented in the provided code.*
* `--disk <file>`:          Path to the disk image file (default: data.x25disk).

### Example

1.  **Generate or obtain your encryption keys/certificates.** (The `xbin25` library likely has tools for this, refer to its documentation).

2.  **Create a mount point directory:**
    ```bash
    mkdir /path/to/mountpoint
    ```

3.  **Mount the filesystem:**
    ```bash
    x25fs --encrypt-cert encrypt.pem --encrypt-key decrypt.pem /path/to/mountpoint
    ```

4.  **Interact with the filesystem** as you would with any other mounted directory.

5.  **Unmount the filesystem:**

    ```bash
    fusermount -u /path/to/mountpoint
    # or
    umount -l /path/to/mountpoint
    ```

##  Security Considerations

* **Key Management:** The security of x25fs heavily relies on the security of your encryption keys.  **Keep your private keys safe!** Loss of the private key will result in the permanent loss of your data.
* **Mount Point Permissions:** x25fs performs checks to ensure the mount point is secure.  Follow best practices for directory permissions to minimize risks.
* **Encryption Algorithm:** The strength of the encryption depends on the algorithms used by the `xbin25` library.  Refer to the `xbin25` documentation for details on its security features.
* **Resource Limits:** x25fs enforces a maximum file size, but it's crucial to monitor disk usage to prevent denial-of-service scenarios.

##  Limitations

* **No Compression:** The `--block-size` option suggests intent for compression, but the provided code does not implement it.
* **Error Handling:** While the code includes error checks, more robust error handling and logging could be beneficial in a production environment.
* **Concurrency:** The code uses mutexes for concurrency control, but further optimization might be possible for high-performance scenarios.
* **Testing:** The provided code does not include unit tests.  Adding tests would improve reliability and maintainability.

##  License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.
