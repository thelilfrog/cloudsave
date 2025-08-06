# CloudSave

The software is still in alpha.

A client/server that allows unsynchronized games (such as emulators, old games, etc.) to be kept up to date on multiple computers.

## Build

You need go1.24

After downloading the go toolchain, just run the script `./build.sh`

## Usage

### Server

The server needs an empty directory. After creating this directory, you need to make a file that contains your credential. The format is "username:password". The server only understand bcrypt password hash for now.

e.g.:
```
test:$2y$10$uULsuyROe3LVdTzFoBH7HO0zhvyKp6CX2FDNl7quXMFYqzitU0kc.
```

The default path to this directory is `/var/lib/cloudsave`, this can be changed with the `-config` argument

### Client

#### Register a game

You can register a game with the verb `add`
```bash
cloudsave add /home/user/gamedata
```

You can also change the name of the registration and add a remote
```bash
cloudsave add -name "My Game" -remote "http://localhost:8080" /home/user/gamedata
```

#### Make an archive of the current state

This is a command line tool, it cannot auto detect changes.
Run this command to start the scan, if needed, the tool will create a new archive

```bash
cloudsave scan
```
#### Send everythings on the server

This will pull and push data to the server.

Note: If multiple computers are pushing to this server, a conflict may be generated. If so, the tool will ask for the version to keep

```bash
cloudsave sync
```
