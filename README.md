# SSH and Sing-box Configuration Tool

This application is a GUI-based tool for managing SSH and Sing-box configurations on routers. It uses the [Fyne](https://fyne.io/) framework for the graphical interface and supports operations like enabling SSH, installing Sing-box, and managing configurations.

## Features

- **Enable SSH:** Activate SSH access on a router by sending commands through HTTP requests.
- **Login via SSH:** Authenticate and execute commands on the router using SSH.
- **Install Sing-box:** Upload and configure Sing-box on the router.
- **Persistent SSH/Sing-box Work after reboot:** Ensure configurations persist through reboots using cron jobs.
- **Router Detection:** Automatically detect the router's IP address in the local subnet.

## Usage

- Just type IP address if router did not autodetect.
- Press "Enable SSH" button.
- Prepare sing-box config and choose your file in "Choose singbox config".
- Press "Install sing-box" 
- Enjoy!
