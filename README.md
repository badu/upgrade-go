# Upgrade Go

Go tool to upgrade Go, for Linux. Why? Because I'm lazy, as I guess you are too...

You must be `sudoer`, since it implies `sudo` without password prompt.

# Sudoer

Run the following commands, where <username> is, yes, your username:

`usermod -aG sudo <username>`

The users’ and groups’ sudo privileges are defined in the /etc/sudoers file. 

Adding the user to this file allows you to grant customized access to the commands and configure custom security policies.

Edit `/etc/sudoers` file while being root and write:

`<username>  ALL=(ALL) NOPASSWD:ALL`

Test you succeeded to become a sudoer:

`sudo ls -la /root`
