# dockerbackup
Allows to test, run, backup and restore automatically docker containers and their volumes, triggered by a cron job.
The executable needs to be in the directory where all .yml config files are to have an easy cron based approach.

# Description
## Config File
Config file is YML based format.
It's a mix between docker compose and docker run syntax. One of the evolutions should be to fully adapt the docker compose .yml file syntax and abandon the docker run syntax.
I have this running for the moment at my server and it is running fine.

			Detach        bool     `yaml:"detach"`
			Privileged    bool     `yaml:"privileged"`
			Init          bool     `yaml:"init"`
			MacMINAddress string   `yaml:"mac-address"`
			Link          string   `yaml:"link"`
			Restart       string   `yaml:"restart"`
			ShmMINsize    string   `yaml:"shm-size"`
			Net           string   `yaml:"net"`
			Volume        []string `yaml:"volume"`
			Device        []string `yaml:"device"`
			Publish       []string `yaml:"publish"`
			Env           []string `yaml:"env"`
			Name          string   `yaml:"name"`
			Githubimage   string   `yaml:"githubimage"`
  
  Name is the final name to appear in docker ps
  Githubimage is the image to pull
  
  ## CLI interface
  The program is CLI based.
      NAME:
       Docker Manager - Fully manage your docker environment

      USAGE:
         rundocker [global options] command [command options] [arguments...]

      VERSION:
         0.0.1

      COMMANDS:
         run, ru      Put your docker file in production, also creates a backup and creates local github image
         test, te     Test your docker run file, create volumes needed
         backup, ba   Backup docker container, volumes, ...
         restore, re  Restore docker container, volumes, ...
         help, h      Shows a list of commands or help for one command

      GLOBAL OPTIONS:
         --all           Look for all docker .yml configfile in the current path (default: false)
         --ymldir value  Define path to look for config files
         --help, -h      show help (default: false)
         --version, -v   print the version (default: false)
         
 ## Autobackup all your docker containers and volumes
 This is an example on how to use the CLI in Cron to automatically backup all your container and volumes defined by the .yml config files in one directory 
 
 	/home/chris/bin/dockerbackupv2 --all --ymldir /home/chris/docker/ backup
 
 
 ## Things to tweak...
 
 You need to use your own repository in dockerhub.
 
     func DetermineOwnGithubImage(githubimage string) string {
        var owngithubimage string
        owngithubimage = "DOCKERHUBIMAGE/backup:" + githubimage
      return owngithubimage
      }
      

You need to define your own log place

    func definelogging() {
      file, err := os.OpenFile("/home/chris/bin/info.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
      if err != nil {
        log.Fatal(err)
        os.Exit(2)
      }
      log.SetOutput(file)
    }
    
    
  You need to define your own log in and password to login in dockerhub
  
        func logintodockerhub() error {
        var executecommand = Dockercommandfunction{
          Commandandarg:     "login --username XXXXX --password XXXXXX",
          Checkresultstring: "",
          Purpose:           "Login to docker hub",
        }
        _, _, err := executedockercommand(executecommand)

        return err
      }
      
      
