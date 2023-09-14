# Factorio auto pause (for multiplayer)

This program is a small utility for Factorio.  
It accomplishes this by automatically pausing the game whenever someone initiates a load or join action, allowing for a smoother and more efficient multiplayer gameplay.

This requires the factorio server to be hosted on Docker.  
Read the factorio server logs from Docker and run the pause command through RCON.


## Required mods

- [Pause commands](https://mods.factorio.com/mod/pause-commands)  
    Add `/pause` and `/unpause` commands.


## Usage

Please refer to [./compose.yml](./compose.yml).
