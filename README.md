# Birdcmd Client

You can use the prebuilt binary `birdcmd` client to connect your device to birdcmd.com server.

The program is written in Go programming language, supports multi platform, you only need to download the prebuilt binary file that matches your machine's architecture, you don't need to install other dependencies.

## Download

Download the most recent prebuilt binary from the release page: https://github.com/birdcmd/birdcmd-go/releases

You can check your machine's architecture with `uname -m` to decide which one you should download.

You could use `wget https://github.com/birdcmd/birdcmd-go/releases/.../birdcmd-xxx.tar.gz` to download from command line.

After downloading, you could decompress it with `tar -xzf birdcmd-xxx.tar.gz`. 

## Deploy

You should see a `birdcmd` binary file (or `birdcmd.exe` on windows).

## Usage

Go to birdcmd.com, click on a tunnel, you should see a blue notebook icon, click it to reveal your `connection` info, in the format of `token:uuid` such as `mybearertoken:my-tunnel-uuid`, copy this text, use it in the following command to start the client:

```
# replace `token:uuid` with the text you copied from your tunnel connection info
# such as ./birdcmd -c mybearertoken:my-tunnel-uuid
./birdcmd -c token:uuid
```

**That's it.**



## Using Systemd

You can use `systemd` in Linux to manage birdcmd client service, including start, stop, running in background and autostart on system boot.

1. **Create birdcmd@.service file**

   Use a text editor (like vim) to create a file `birdcmd@.service` in `/etc/system/systemd` folder to configure birdcmd client service.

   ```
   $ sudo vim /etc/systemd/system/birdcmd@.service
   ```

   Write the following to it:

   ```
   [Unit]
   Description=BirdCMD Service
   After=network.target
   
   [Service]
   Type=idle
   User=your-user-name
   Restart=on-failure
   RestartSec=60s
   ExecStart=/path/to/birdcmd -c %i
   WorkingDirectory=/path/to/birdcmd
   
   [Install]
   WantedBy=multi-user.target
   ```

   Remember to **change** `your-user-name` to your actual user's name (such as `pi` on a Raspberry Pi) if you need to run program with permissions, because systemd creates a temporary user with minimal permissions.

   Also **change** `/path/to/birdcmd` to your actual `birdcmd` client's path.

2. **Start/Stop**

   To start/stop the service:

   ```bash
   systemctl <start|stop> birdcmd@<Unit Name>
   ```

   The `<Unit Name>` is your tunnel's connection info, if your tunnel's connection info is: `mybearertoken:my-tunnel-uuid`, you can run:

   ```bash
   systemctl start birdcmd@mybearertoken:my-tunnel-uuid
   ```

3. **Check Status**

   To check the status of your birdcmd client:

   ```bash
   systemctl status birdcmd@<Unit Name>
   ```

   For example, check the connection info with `mybearertoken:my-tunnel-uuid`:

   ```bash
   systemctl status birdcmd@mybearertoken:my-tunnel-uuid
   ```

   **Do not** start one tunnel more than once! Running `systemctl start` multiple times is safe, but after configuring systemd, **do not** manually run `birdcmd -c <params>`  to start a tunnel client.

   If you forgot which tunnels are started, you can check with:

   ```bash
   systemctl list-units "birdcmd@*"
   ```

4. **Check the Logs of the Birdcmd Client**

   You can check the logs with:

   ```bash
   journalctl -u birdcmd@<Unit Name>
   ```

   For example:

   ```bash
   journalctl -u birdcmd@mybearertoken:my-tunnel-uuid
   ```

5. **Auto Start on System Boot**

   You can set the birdcmd systemd service to auto start on system boot:

   ```bash
   systemctl <enable|disable> birdcmd@<Unit Name>
   ```

   For example, to enable autostart on system boot:

   ```bash
   systemctl enable birdcmd@mybearertoken:my-tunnel-uuid
   ```

   To disable autostart:

   ```bash
   systemctl disable birdcmd@mybearertoken:my-tunnel-uuid
   ```

   To check the status:

   ```bash
   systemctl status birdcmd@mybearertoken:my-tunnel-uuid
   ```

   You should see: `Loaded: loaded (/etc/systemd/system/birdcmd@.service; enabled; vendor preset: enabled)` has a `enable` text after enabling auto start.

   If you forget which tunnels are set to auto start on system boot, you can use:

   ```bash
   systemctl list-units --all "birdcmd@*"
   ```

   It will list stopped services with the extra `--all`.

