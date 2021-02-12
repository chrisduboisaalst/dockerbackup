package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

var (
	Normal       = Teal
	Warn         = Yellow
	Fata         = Red
	Normaloutput = Green
)

var (
	Black   = Color("\033[1;30m%s\033[0m")
	Red     = Color("\033[1;31m%s\033[0m")
	Green   = Color("\033[1;32m%s\033[0m")
	Yellow  = Color("\033[1;33m%s\033[0m")
	Purple  = Color("\033[1;34m%s\033[0m")
	Magenta = Color("\033[1;35m%s\033[0m")
	Teal    = Color("\033[1;36m%s\033[0m")
	White   = Color("\033[1;37m%s\033[0m")
)

func Color(colorString string) func(...interface{}) string {
	sprint := func(args ...interface{}) string {
		return fmt.Sprintf(colorString,
			fmt.Sprint(args...))
	}
	return sprint
}

var app = cli.NewApp()

type backuptype int

const (
	Volume = iota
	Image
)

type positiontype int

const (
	Begin = iota
	Middle
	End
	BeginEnd
)

type blocktype int

const (
	Feedback = iota
	Feedbackoutput
	Warning
	Critical
)

type printblock struct {
	position  positiontype
	block     blocktype
	blockname string
	text      string
}

type DockerArgs interface {
	toarg() []string
}

type Config struct {
	Server struct {
		Port string `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`
	Database struct {
		Username string `yaml:"user"`
		Password string `yaml:"pass"`
	} `yaml:"database"`
}

type Runimage struct {
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
}

type Dockerconfig struct {
	Testimage      Runimage `yaml:"runimage"`
	Configfilename string
	Configfiledir  string
}

type Dockercommandfunction struct {
	Commandandarg      string
	Commandandargslice []string
	Checkresultstring  string
	Purpose            string
}

type Backupfileinfo struct {
	Filename        string
	Whichbackuptype backuptype
	DatetoRestore   string
}

func (c Runimage) toarg() []string {
	var args []string
	var argcommand string
	var argvalue string
	var printtoargs printblock

	cvalue := reflect.ValueOf(c)
	if cvalue.Kind() == reflect.Ptr {
		cvalue = cvalue.Elem()
	}
	ctype := cvalue.Type()

	// initiate run docker command

	argcommand = "run"
	argvalue = ""
	addargsfordocker(&args, argcommand, argvalue)

	for i := 0; i < cvalue.NumField(); i++ {
		fieldvalue := cvalue.Field(i)
		fieldtype := ctype.Field(i)
		switch fieldvalue.Kind() {
		case reflect.Struct:
			println("Error, no struct expected")
		case reflect.Bool:
			argcommand = "--" + fieldtype.Name
			argvalue = ""
			argbool := fieldvalue.Interface().(bool)
			if argbool {
				addargsfordocker(&args, argcommand, argvalue)
			}
		case reflect.Slice:
			argcommand = "--" + fieldtype.Name
			for _, element := range fieldvalue.Interface().([]string) {
				argvalue = element
				addargsfordocker(&args, argcommand, argvalue)
			}
		default:
			argvalue = fieldvalue.Interface().(string)
			if fieldtype.Name == "Githubimage" {
				argcommand = ""
				addargsfordocker(&args, argcommand, argvalue)
			} else {
				if argvalue != "" {
					argcommand = "--" + fieldtype.Name
					addargsfordocker(&args, argcommand, argvalue)
				}
			}

		}

	}

	printtoargs = printblock{
		position:  BeginEnd,
		block:     Feedbackoutput,
		blockname: "Read config file to args",
		text:      strings.Join(args, " "),
	}
	printtoargs.bprint()

	for _, dockerfullcommand := range args {
		log.Println("Dockerfullcommand: %v\n", dockerfullcommand)
	}

	return args
}

func (c printblock) bprint() {
	var openblock string
	var closeblock string

	colortext := Normal

	switch c.block {
	case Feedback:
		colortext = Normal
	case Feedbackoutput:
		colortext = Normaloutput
	case Warning:
		colortext = Warn
	case Critical:
		colortext = Fata
	}
	switch c.position {
	case Begin:
		openblock = "**** " + c.blockname
		closeblock = ""
	case Middle:
		openblock = ""
		closeblock = ""
	case End:
		openblock = ""
		closeblock = "----"
	case BeginEnd:
		openblock = "**** " + c.blockname
		closeblock = "----"
	}

	if openblock != "" {
		fmt.Println()
		fmt.Println(openblock)
		fmt.Println()
	}
	fmt.Println(colortext("     " + c.text))
	if closeblock != "" {
		fmt.Println(closeblock)
	}

}

func executedockercommand(dockercommandstruct Dockercommandfunction) (test bool, outputstring string, err error) {
	var args []string
	var app string
	var printexecutedockercommand printblock
	var out bytes.Buffer
	var stderr bytes.Buffer

	app = "docker"

	// Prepare command structure for docker
	if dockercommandstruct.Commandandarg == "" {
		args = dockercommandstruct.Commandandargslice
	} else {
		args = strings.Split(dockercommandstruct.Commandandarg, " ")
	}

	// Print to stdout what is going to be done
	printexecutedockercommand = printblock{
		position:  Begin,
		blockname: dockercommandstruct.Purpose,
		block:     Feedback,
		text:      "Docker command : docker " + strings.Join(args, " "),
	}
	printexecutedockercommand.bprint()

	// Execute Docker command

	cmd := exec.Command(app, args...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()

	// Interprete results of Docker command

	if err != nil {

		printexecutedockercommand = printblock{
			position:  End,
			blockname: dockercommandstruct.Purpose,
			block:     Critical,
			text:      fmt.Sprint(err) + ": " + stderr.String() + " output:" + out.String(),
		}

		printexecutedockercommand.bprint()

		test := false
		outputstring := out.String()

		return test, outputstring, err

	} else {
		if dockercommandstruct.Checkresultstring == "" {

			printexecutedockercommand = printblock{
				position:  End,
				blockname: dockercommandstruct.Purpose,
				block:     Feedbackoutput,
				text:      "Output :" + out.String(),
			}

			printexecutedockercommand.bprint()

			test := false
			outputstring := out.String()
			return test, outputstring, err
		} else {
			if strings.Contains(strings.ToUpper(out.String()), strings.ToUpper(dockercommandstruct.Checkresultstring)) {

				printexecutedockercommand = printblock{
					position:  End,
					blockname: dockercommandstruct.Purpose,
					block:     Feedbackoutput,
					text:      "Docker test successful :" + dockercommandstruct.Commandandarg,
				}
				printexecutedockercommand.bprint()

				test := true
				outputstring := out.String()
				return test, outputstring, err
			} else {

				printexecutedockercommand = printblock{
					position:  End,
					blockname: dockercommandstruct.Purpose,
					block:     Warning,
					text:      "Docker test not successful :" + dockercommandstruct.Commandandarg,
				}
				printexecutedockercommand.bprint()

				test := false
				outputstring := out.String()
				return test, outputstring, err
			}

		}
	}
}

func killrunningcontainer(runningcontainer string) error {
	var executecommand Dockercommandfunction
	executecommand = Dockercommandfunction{
		Commandandarg: "container stop " + runningcontainer,
		Purpose:       "Kill running container" + runningcontainer + " (Start)",
	}
	_, _, err := executedockercommand(executecommand)

	executecommand = Dockercommandfunction{
		Commandandarg: "container rm " + runningcontainer,
		Purpose:       "Kill running container" + runningcontainer + " (End)",
	}
	_, _, err = executedockercommand(executecommand)
	return err
}

func containerrunning(runningcontainer string) bool {
	var executecommand = Dockercommandfunction{
		Commandandarg:     "container list -a --filter name=" + runningcontainer,
		Checkresultstring: runningcontainer,
		Purpose:           "Check if container: " + runningcontainer + " is running",
	}
	dockertest, _, err := executedockercommand(executecommand)

	if err != nil {
		return false
	} else {
		return dockertest
	}
}

func createvolume(volumetocreate string) error {
	var executecommand = Dockercommandfunction{
		Commandandarg:     "volume create " + volumetocreate,
		Checkresultstring: "",
		Purpose:           "Create dockervolume : " + volumetocreate,
	}
	_, _, err := executedockercommand(executecommand)

	return err
}

func createnetwork(networktocreate string) error {
	var executecommand = Dockercommandfunction{
		Commandandarg:     "network create " + networktocreate,
		Checkresultstring: "",
		Purpose:           "Create docker netwerk : " + networktocreate,
	}
	_, _, err := executedockercommand(executecommand)

	return err
}

func volumeexists(createdvolume string) bool {
	var executecommand = Dockercommandfunction{
		Commandandarg:     "volume ls --filter name=" + createdvolume,
		Checkresultstring: createdvolume,
		Purpose:           "Check if docker volume: " + createdvolume + " exists",
	}
	dockertest, _, err := executedockercommand(executecommand)

	if err != nil {
		return false
	} else {
		return dockertest
	}
}

func networkexists(network string) bool {
	var executecommand = Dockercommandfunction{
		Commandandarg:     "network ls --filter name=" + network,
		Checkresultstring: network,
		Purpose:           "Check if docker network: " + network + " exists",
	}
	dockertest, _, err := executedockercommand(executecommand)

	if err != nil {
		return false
	} else {
		return dockertest
	}
}

func addargsfordocker(c *[]string, command string, value string) {
	//fullstring := strings.ToLower(command + " " + value)
	//fullstring = strings.TrimSpace(fullstring)
	//*c = append(*c, fullstring)
	if command != "" {
		commandtolower := strings.ToLower(command)
		commandtolower = strings.ReplaceAll(commandtolower, "min", "-")
		*c = append(*c, commandtolower)
	}

	if value != "" {
		*c = append(*c, value)
	}
}

func (c Dockerconfig) toarg() []string {
	var args []string
	var argcommand string
	var argvalue string

	cvalue := reflect.ValueOf(c)
	if cvalue.Kind() == reflect.Ptr {
		cvalue = cvalue.Elem()
	}
	ctype := cvalue.Type()

	for i := 0; i < cvalue.NumField(); i++ {
		fieldvalue := cvalue.Field(i)
		fieldtype := ctype.Field(i)
		if fieldvalue.Kind() != reflect.Struct {
			argcommand = "- " + fieldtype.Name
			argvalue = " " + fieldvalue.Interface().(string)
			args = append(args, argcommand+argvalue)
		} else {
			fieldvalue.MethodByName("toarg").Call([]reflect.Value{})
		}
	}

	for _, dockerfullcommand := range args {
		fmt.Println("Dockerfullcommand: %p", dockerfullcommand)
	}

	return args
}

func whichfile(c *cli.Context) (string, string, error) {
	if (c.Args().Len()) == 0 {
		return "", "", errors.New("No argument given !")
	}

	configfilename := c.Args().Get(0)
	if !strings.Contains(configfilename, ".yml") {
		return "", "", errors.New("No .yml file specified")
	}

	ymldir := c.String("ymldir")

	return configfilename, ymldir, nil

}

func findallymlfiles(ymldir string) ([]string, error) {
	if ymldir != "" {
		err := os.Chdir(ymldir)
		if err != nil {
			readconfigfileprint := printblock{
				position:  BeginEnd,
				block:     Critical,
				blockname: "Impossible change working directory",
				text:      "Cannot change to ..." + ymldir,
			}
			readconfigfileprint.bprint()
			return nil, err
		}
	}
	matches, err := filepath.Glob("*.yml")
	if err != nil {
		readconfigfileprint := printblock{
			position:  BeginEnd,
			block:     Critical,
			blockname: "Impossible to find *.yml file",
			text:      "Not possible to look for *.yml file" + err.Error(),
		}
		readconfigfileprint.bprint()
	}
	if len(matches) == 0 {
		readconfigfileprint := printblock{
			position:  BeginEnd,
			block:     Critical,
			blockname: "Looking for *.yml file",
			text:      "No docker config files found",
		}
		readconfigfileprint.bprint()
	}
	return matches, err
}

func readFile(cfg *Dockerconfig) {

	f, err := os.Open(cfg.Configfiledir + cfg.Configfilename)

	if err != nil {
		readconfigfileprint := printblock{
			position:  BeginEnd,
			block:     Critical,
			blockname: "Read " + cfg.Configfilename,
			text:      "Not possible to open and read file: " + err.Error(),
		}
		readconfigfileprint.bprint()
		os.Exit(2)
	} else {
		readconfigfileprint := printblock{
			position:  Begin,
			block:     Feedbackoutput,
			blockname: "Open " + cfg.Configfilename,
			text:      "File opened and read",
		}
		readconfigfileprint.bprint()
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		readconfigfileprint := printblock{
			position:  End,
			block:     Critical,
			blockname: "Open " + cfg.Configfilename,
			text:      "Not possible to convert yml file to internal structure :" + err.Error(),
		}
		readconfigfileprint.bprint()
		os.Exit(2)
	} else {
		readconfigfileprint := printblock{
			position:  End,
			block:     Feedbackoutput,
			blockname: "Open " + cfg.Configfilename,
			text:      "Config.yml converted into internal structure ",
		}
		readconfigfileprint.bprint()
	}
}

func removeimage(owngithubimage string) error {
	var executecommand = Dockercommandfunction{
		Commandandarg:     "image rm " + owngithubimage + " -f",
		Checkresultstring: "",
		Purpose:           "remove existing local image: " + owngithubimage,
	}
	_, _, err := executedockercommand(executecommand)

	return err
}

func removevolume(volumename string) error {

	var executecommand = Dockercommandfunction{
		Commandandarg:     "system prune -f",
		Checkresultstring: "",
		Purpose:           "Prune all dangling images, ... " + volumename,
	}
	_, _, err := executedockercommand(executecommand)

	executecommand = Dockercommandfunction{
		Commandandarg:     "volume rm -f " + volumename,
		Checkresultstring: "",
		Purpose:           "force remove existing volume: " + volumename,
	}
	_, _, err = executedockercommand(executecommand)

	return err
}

func emptyvolume(volumename string) error {

	//Make sure that the image that is using this volume is stopped
	// Identify mountpoint
	var executecommand = Dockercommandfunction{
		Commandandarg:     "volume inspect " + volumename,
		Checkresultstring: "",
		Purpose:           "Find Mountpoint of this volume : " + volumename,
	}

	_, outputstring, err := executedockercommand(executecommand)
	outputstring = strings.ReplaceAll(outputstring, "[\n", "")
	outputstring = strings.ReplaceAll(outputstring, "]\n", "")

	outputbytes := []byte(outputstring)
	var raw map[string]interface{}
	if err = json.Unmarshal(outputbytes, &raw); err != nil {
		return err
	}

	//empty files in volume

	mountpoint := raw["Mountpoint"].(string)

	safetodelete := strings.Contains(mountpoint, volumename) && strings.Contains(mountpoint, "docker")
	if safetodelete {
		fmt.Println("Removing files in volume :", volumename)
		err = os.RemoveAll(mountpoint)
	} else {
		fmt.Println("Not safe to remove files in volume :", volumename)
	}

	return err
}

func logintodockerhub() error {
	var executecommand = Dockercommandfunction{
		Commandandarg:     "login --username XXXXX --password XXXXXX",
		Checkresultstring: "",
		Purpose:           "Login to docker hub",
	}
	_, _, err := executedockercommand(executecommand)

	return err
}

func logoutfromdockerhub() error {
	var executecommand = Dockercommandfunction{
		Commandandarg:     "logout",
		Checkresultstring: "",
		Purpose:           "Logout from docker hub",
	}
	_, _, err := executedockercommand(executecommand)

	return err
}
func pushimagetoowngithub(owngithubimage string) error {
	var executecommand = Dockercommandfunction{
		Commandandarg:     "push " + owngithubimage,
		Checkresultstring: "",
		Purpose:           "Push image to own github  : " + owngithubimage,
	}
	_, _, err := executedockercommand(executecommand)

	return err
}

func tagimage(githubimage string, owngithubimage string) error {
	var executecommand = Dockercommandfunction{
		Commandandarg:     "tag " + githubimage + " " + owngithubimage,
		Checkresultstring: "",
		Purpose:           "Tag volume for own github  : " + owngithubimage,
	}
	_, _, err := executedockercommand(executecommand)

	return err
}

func checkimage(owngithubimage string) (check bool, err error) {

	repository := strings.Split(owngithubimage, ":")

	var executecommand = Dockercommandfunction{
		Commandandarg:     "image ls -a --filter reference=" + owngithubimage,
		Checkresultstring: repository[0],
		Purpose:           "Check if local image: " + repository[0] + " exists",
	}
	dockertest, _, err := executedockercommand(executecommand)

	if err != nil {
		return false, err
	} else {
		return dockertest, err
	}
}

func DetermineOwnGithubImage(githubimage string) string {
	var owngithubimage string
	owngithubimage = "DOCKERHUBIMAGE/backup:" + githubimage

	return owngithubimage
}

func TargetBackupFullPath() string {
	var backupdir string
	backupdir = "/mnt/raid/dockerbackup/"

	return backupdir
}

func imagetoowngithub(githubimage string, newimagename string) (owngithubimagevar string, errorvar error) {
	var owngithubimage string
	owngithubimage = DetermineOwnGithubImage(newimagename)

	err := tagimage(githubimage, owngithubimage)
	if err != nil {
		return owngithubimage, err
	}

	err = logintodockerhub()
	if err != nil {
		return owngithubimage, err
	}

	err = pushimagetoowngithub(owngithubimage)
	if err != nil {
		return owngithubimage, err
	}

	err = logoutfromdockerhub()
	if err != nil {
		return owngithubimage, err
	}

	return owngithubimage, nil
}

func LaunchTestDocker(c *cli.Context) error {

	_, err := TestDocker(c)
	return err
}

func PrepareDataStructures(cfg *Dockerconfig) (returnrunimage Runimage, returnargs []string) {
	var dockercfg Runimage
	var args []string
	// Read config file
	readFile(cfg)

	// Convert config file to arg
	dockercfg = cfg.Testimage
	args = dockercfg.toarg()

	return dockercfg, args
}

func TestDocker(c *cli.Context) (configdata Runimage, returnerr error) {
	var cfg Dockerconfig
	var dockertestcfg Runimage
	var args []string
	var err error
	var dockercommand Dockercommandfunction
	var volume []string
	var ymldir string
	var configfilename string

	configfilename, ymldir, err = whichfile(c)
	if err != nil {
		return dockertestcfg, err
	}

	cfg.Configfilename = configfilename
	cfg.Configfiledir = ymldir

	dockertestcfg, args = PrepareDataStructures(&cfg)

	// Test if container is already running
	if containerrunning(dockertestcfg.Name) {
		err = killrunningcontainer(dockertestcfg.Name)
	}

	// Test if volumes have been created, create them if needed

	for _, element := range dockertestcfg.Volume {
		volume = strings.Split(element, ":")
		if !strings.Contains(volume[0], "/") {
			if !volumeexists(volume[0]) {
				createvolume(volume[0])
			}
		}
	}

	// Test if network has been created, create them if needed
	if dockertestcfg.Net != "" {
		if !networkexists(dockertestcfg.Net) {
			createnetwork(dockertestcfg.Net)
		}

	}

	// Run container
	dockercommand = Dockercommandfunction{
		Commandandargslice: args,
		Purpose:            "Run Container : " + dockertestcfg.Name,
	}
	_, _, err = executedockercommand(dockercommand)

	return dockertestcfg, err
}

func StartContainer(containername string) error {

	// Start new container
	dockercommand := Dockercommandfunction{
		Commandandarg:     "start " + containername,
		Checkresultstring: "",
		Purpose:           "Start stopped container :" + containername,
	}
	_, _, err := executedockercommand(dockercommand)

	return err
}
func RunContainer(dockerconfig Runimage) error {

	args := dockerconfig.toarg()

	// Run new container
	dockercommand := Dockercommandfunction{
		Commandandargslice: args,
		Purpose:            "Run Container : " + dockerconfig.Name,
	}
	_, _, err := executedockercommand(dockercommand)

	return err
}

func Runcontainerfromownimage(olddockerconfig Runimage, newdockerconfig Runimage) error {

	var err error

	// stop and delete old running container
	if containerrunning(olddockerconfig.Name) {
		err = killrunningcontainer(olddockerconfig.Name)
		if err != nil {
			return err
		}
	}

	// delete old image pulled from github www
	exists, err := checkimage(olddockerconfig.Githubimage)
	if err != nil {
		return err
	}
	if exists {
		err = removeimage(olddockerconfig.Githubimage)
		if err != nil {
			return err
		}
	}

	// delete old image pulled from own github

	exists, err = checkimage(newdockerconfig.Githubimage)
	if err != nil {
		return err
	}
	if exists {
		err = removeimage(newdockerconfig.Githubimage)
		if err != nil {
			return err
		}
	}
	//login to docker cloud to be able to pull new image
	err = logintodockerhub()
	if err != nil {
		return err
	}

	err = RunContainer(newdockerconfig)

	//logout from docker cloud
	err = logoutfromdockerhub()

	return err
}

func RunDocker(c *cli.Context) error {
	var dockerconfig Runimage
	var newdockerconfig Runimage
	var err error
	var owndockerimage string

	dockerconfig, err = TestDocker(c)
	if err != nil {
		return err
	}

	newdockerconfig = dockerconfig

	// Push image to own repository
	owndockerimage, err = imagetoowngithub(dockerconfig.Githubimage, dockerconfig.Name)
	if err != nil {
		return err
	}

	newdockerconfig.Githubimage = owndockerimage
	// Delete container using github image and replace by container running own image

	err = Runcontainerfromownimage(dockerconfig, newdockerconfig)
	if err != nil {
		return err
	}

	return nil
}

func StopRunningContainer(githubimage string) error {
	var executecommand = Dockercommandfunction{
		Commandandarg:     "stop " + githubimage,
		Checkresultstring: "",
		Purpose:           "Stop container  : " + githubimage,
	}
	_, _, err := executedockercommand(executecommand)

	return err
}

func CreateBackupContainer(containername string) (mybackupfile string, mybackupfilefullpath string, myerror error) {

	var backupfilefullpath string
	var backupfile string
	var err error
	var volumeinfo Backupfileinfo

	volumeinfo.Filename = containername
	volumeinfo.Whichbackuptype = Image

	_, backupfile, backupfilefullpath, err = GiveBackupFileInfo(volumeinfo)
	if err != nil {
		return "", "", err
	}

	//Create temp image in local repository
	var executecommand = Dockercommandfunction{
		Commandandarg:     "commit -p " + containername + " " + containername + "_temp",
		Checkresultstring: "",
		Purpose:           "Backup container " + containername,
	}
	_, _, err = executedockercommand(executecommand)

	if err != nil {
		return "", "", err
	}

	// Convert image in tar file
	executecommand = Dockercommandfunction{
		Commandandarg:     "save -o " + backupfilefullpath + " " + containername + "_temp",
		Checkresultstring: "",
		Purpose:           "Write container " + containername,
	}
	_, _, err = executedockercommand(executecommand)

	if err != nil {
		return "", "", err
	}

	// Remove temp image

	executecommand = Dockercommandfunction{
		Commandandarg:     "image rm " + containername + "_temp",
		Checkresultstring: "",
		Purpose:           "Remove Temp backup container : " + containername + "_temp",
	}
	_, _, err = executedockercommand(executecommand)

	if err != nil {
		return "", "", err
	}

	return backupfile, backupfilefullpath, err

}

func GiveBackupFileInfo(VolumeContainerinfo Backupfileinfo) (myworkdir string, mybackupfile string, mybackupfilefullpath string, myerror error) {
	var err error
	var backupfilefullpath string
	var backupfile string
	var backupsuffix string
	var filedate string

	switch VolumeContainerinfo.DatetoRestore {
	case "":
		dt := time.Now()
		filedate = dt.Format("02-01-2006")
	default:
		filedate = VolumeContainerinfo.DatetoRestore
	}

	workdir, err := os.Getwd()
	if err != nil {
		return "", "", "", err
	}

	switch VolumeContainerinfo.Whichbackuptype {
	case Volume:
		backupsuffix = "_volume_archive_"
	case Image:
		backupsuffix = "_image_archive_"
	}

	backupfile = VolumeContainerinfo.Filename + backupsuffix + filedate + ".tar"
	backupfilefullpath = workdir + "/" + backupfile

	return workdir, backupfile, backupfilefullpath, err
}

func CreateBackupVolume(volumename string) (mybackupfile string, mybackupfilefullpath string, myerror error) {

	var err error
	var backupfilefullpath string
	var backupfile string
	var workdir string
	var volumeinfo Backupfileinfo

	volumeinfo.Filename = volumename
	volumeinfo.Whichbackuptype = Volume

	workdir, backupfile, backupfilefullpath, err = GiveBackupFileInfo(volumeinfo)
	if err != nil {
		return "", "", err
	}

	var executecommand = Dockercommandfunction{
		Commandandarg: "run -v " + volumename + ":/volume -v " + workdir +
			":/backup alpine tar -cf /backup/" + backupfile + " -C /volume ./",
		Checkresultstring: "",
		Purpose:           "Create backup volume " + volumename,
	}

	_, _, err = executedockercommand(executecommand)

	if err != nil {
		return backupfile, backupfilefullpath, err
	}

	return backupfile, backupfilefullpath, nil
}

func CopyConfigFile(targetfilefullpath string, sourcefile string, ymldir string) (err error) {
	var printexecutedockercommand printblock

	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			printexecutedockercommand = printblock{
				position:  End,
				blockname: "Copy configfile to backupdir",
				block:     Critical,
				text:      fmt.Sprint(err),
			}
		} else {
			printexecutedockercommand = printblock{
				position:  End,
				blockname: "Copy configfile to backupdir",
				block:     Feedbackoutput,
				text:      "io.Copy  " + sourcefile + " to " + targetfilefullpath,
			}
		}
		printexecutedockercommand.bprint()
	}()

	printexecutedockercommand = printblock{
		position:  Begin,
		blockname: "Copy configfile to backupdir",
		block:     Feedback,
		text:      "io.Copy  " + sourcefile + " to " + targetfilefullpath,
	}
	printexecutedockercommand.bprint()
	from, err := os.Open(ymldir + sourcefile) // "./"
	if err != nil {
		panic(err)
	}
	defer from.Close()

	to, err := os.OpenFile(targetfilefullpath+"/"+sourcefile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		panic(err)
	}

	return
}

func MoveBackup(origfilefullpath string, targetfilefullpath string) error {

	var out bytes.Buffer
	var stderr bytes.Buffer

	printexecutedockercommand := printblock{
		position:  Begin,
		blockname: "Move file to backupdir",
		block:     Feedback,
		text:      "mv  " + origfilefullpath + " " + targetfilefullpath,
	}
	printexecutedockercommand.bprint()

	cmd := exec.Command("mv", origfilefullpath, targetfilefullpath)
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		printexecutedockercommand = printblock{
			position:  End,
			blockname: "Move backupfile to backupdir",
			block:     Critical,
			text:      fmt.Sprint(err) + ": " + stderr.String() + " output:" + out.String(),
		}
	} else {
		printexecutedockercommand = printblock{
			position:  End,
			blockname: "Move backupfile to backupdir",
			block:     Feedbackoutput,
			text:      "mv  " + origfilefullpath + " " + targetfilefullpath,
		}
	}
	printexecutedockercommand.bprint()
	return err
}

func BackupContainer1by1(c *cli.Context, configfilename string, ymldir string) error {
	var dockerconfig Runimage
	var owngithubdockerconfig Runimage
	var cfg Dockerconfig
	var err error
	var backupfilefullpath, backupfile string

	cfg.Configfilename = configfilename
	cfg.Configfiledir = ymldir

	dockerconfig, _ = PrepareDataStructures(&cfg)
	owngithubdockerconfig = dockerconfig
	owngithubdockerconfig.Githubimage = DetermineOwnGithubImage(dockerconfig.Name)

	// check if final container is running
	exists := containerrunning(owngithubdockerconfig.Name)

	if !exists {
		return errors.New("Container is not running, make sure you use run before trying to backup")
	}

	//check if final image is running

	exists, err = checkimage(owngithubdockerconfig.Githubimage)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("Container is running, but local image is not used make sure you use run before trying to backup")
	}

	//stop container to avoid changes in volume

	err = StopRunningContainer(owngithubdockerconfig.Name)
	if err != nil {
		return err
	}

	// create backup container and move to the backup directory

	backupfile, backupfilefullpath, err = CreateBackupContainer(owngithubdockerconfig.Name)
	if err != nil {
		return err
	}
	err = MoveBackup(backupfilefullpath, TargetBackupFullPath()+backupfile)
	if err != nil {
		return err
	}

	// copy config file
	err = CopyConfigFile(TargetBackupFullPath(), configfilename, ymldir)
	if err != nil {
		return err
	}
	//backup volumes...

	for _, element := range owngithubdockerconfig.Volume {
		volume := strings.Split(element, ":")
		if !strings.Contains(volume[0], "/") {
			if !volumeexists(volume[0]) {
				return errors.New("Volume does not exist, please run the image before you try to backup")
			}
			backupfile, backupfilefullpath, err = CreateBackupVolume(volume[0])
			if err != nil {
				return err
			}
			err = MoveBackup(backupfilefullpath, TargetBackupFullPath()+backupfile)
			if err != nil {
				return err
			}
		}
	}

	// Restart Container

	err = StartContainer(owngithubdockerconfig.Name)
	if err != nil {
		return err
	}

	return nil
}

func BackupContainer(c *cli.Context) error {

	fmt.Println("Backup docker container")

	if c.Bool("all") {
		fmt.Println("Backup all config files")
		allconfigfiles, err := findallymlfiles(c.String("ymldir"))
		if err != nil {
			return err
		}
		for _, configfilename := range allconfigfiles {
			err = BackupContainer1by1(c, configfilename, c.String("ymldir"))
			if err != nil {
				fmt.Println("Failed to backup :", configfilename)
			} else {
				fmt.Println("Successfully backed up :", configfilename)
			}
		}

	} else {
		configfilename, ymldir, err := whichfile(c)
		fmt.Println("YMLDIR :", ymldir)
		if err != nil {
			return err
		}
		err = BackupContainer1by1(c, configfilename, ymldir)
		if err != nil {
			return err
		}
	}

	return nil

}

func RestoreBackupVolume(volumename string, backupvolumedir string, restorevolumetar string) (err error) {

	var executecommand = Dockercommandfunction{
		//Commandandargslice: []string{`run`, `-v`, volumename + `:/volume`, `-v`, backupvolumedir + `:/backup`, `alpine`, `/bin/sh -c "rm -rf /volume/* "`},
		Commandandarg: `run -v ` + volumename + `:/volume -v ` + backupvolumedir +
			`:/backup alpine tar -C /volume/ -xf /backup/` + restorevolumetar,
		Checkresultstring: "",
		Purpose:           "Restore backupvolume : " + restorevolumetar,
	}

	_, _, err = executedockercommand(executecommand)

	return
}

func ReadArgs(c *cli.Context) (configfilename string, backupdate string, err error) {

	if (c.Args().Len()) < 2 {
		return "", "", errors.New("2 arguments needed : configfilename, DD-MM-YYYY")
	}

	configfilename = c.Args().Get(0)
	if !strings.Contains(configfilename, ".yml") {
		return "", "", errors.New("No .yml file specified")
	}

	backupdate = c.Args().Get(1)
	fmt.Println("Backupdate :", backupdate)

	_, err = time.Parse("02-01-2006", backupdate)
	if err != nil {
		fmt.Println("Error : ", err)
		return configfilename, "", errors.New("Backupdate in wrong format")
	}

	return
}

func RestoreContainer(c *cli.Context) (err error) {

	var cfg Dockerconfig
	var volumeinfo Backupfileinfo

	fmt.Println("Restore docker container")

	var configfilename string
	var backupdate string

	configfilename, backupdate, err = ReadArgs(c)
	if err != nil {
		return err
	}

	cfg.Configfilename = configfilename

	dockerconfig, _ := PrepareDataStructures(&cfg)
	owngithubdockerconfig := dockerconfig
	owngithubdockerconfig.Githubimage = DetermineOwnGithubImage(dockerconfig.Name)

	// 2 possible ways of doing this
	// or we reload the image that we have pushed to the docker cloud
	// or we reload the image that we have saved as tar
	// as we are in the methodology of changing the downloaded image to our own image
	// it seems more logical to use the image that we have downloaded in our own docker (cloud) environment

	//stop current container

	err = StopRunningContainer(owngithubdockerconfig.Name)
	if err != nil {
		fmt.Sprintln("Container " + owngithubdockerconfig.Name + " is not running, this is not necessarily an issue ")
	}

	// remove local image

	exists, err := checkimage(owngithubdockerconfig.Githubimage)
	if err != nil {
		fmt.Sprintln("Image " + owngithubdockerconfig.Githubimage + " does not exist, this is not necessarily an issue ")
	}
	if exists {
		err = removeimage(owngithubdockerconfig.Githubimage)
		if err != nil {
			return err
		}
	}

	//load backup image
	// this will be loaded automatically once we run the docker file
	// otherwise docker load -i ${CONTAINER}.tar should do the trick

	// load backup volumes

	for _, element := range owngithubdockerconfig.Volume {
		volume := strings.Split(element, ":")
		if !strings.Contains(volume[0], "/") {
			if !volumeexists(volume[0]) {
				fmt.Println("Volume " + volume[0] + " does not exist, this is not necessarily an issue, the volume will be restored")
			} else {
				err = removevolume(volume[0])
				if err != nil {
					fmt.Println("Removing volume to avoid issues when restoring volume")
					return err
				}
			}
			volumeinfo.Filename = volume[0]
			volumeinfo.Whichbackuptype = Volume
			volumeinfo.DatetoRestore = backupdate
			_, backupfile, _, err := GiveBackupFileInfo(volumeinfo)
			if err != nil {
				return err
			}
			err = RestoreBackupVolume(volume[0], TargetBackupFullPath(), backupfile)
			if err != nil {
				return err
			}

		}
	}

	// run_image
	//login to docker cloud to be able to pull new image
	err = logintodockerhub()
	if err != nil {
		return err
	}

	err = RunContainer(owngithubdockerconfig)
	if err != nil {
		return err
	}
	//logout from docker cloud
	err = logoutfromdockerhub()
	if err != nil {
		return err
	}

	return nil
}

func defineapp() {
	//Define Flags
	allconfigfilesflag := cli.BoolFlag{
		Name:  "all",
		Usage: "Look for all docker .yml configfile in the current path",
	}
	configfilesdirflag := cli.StringFlag{
		Name:  "ymldir",
		Usage: "Define path to look for config files",
	}

	//Define Commands
	runcommand := cli.Command{
		Name:    "run",
		Aliases: []string{"ru"},
		Usage:   "Put your docker file in production, also creates a backup and creates local github image",
		Action:  RunDocker,
	}

	testcommand := cli.Command{
		Name:    "test",
		Aliases: []string{"te"},
		Usage:   "Test your docker run file, create volumes needed",
		Action:  LaunchTestDocker,
	}
	backupcommand := cli.Command{
		Name:    "backup",
		Aliases: []string{"ba"},
		Usage:   "Backup docker container, volumes, ...",
		Action:  BackupContainer,
	}

	restorecommand := cli.Command{
		Name:    "restore",
		Aliases: []string{"re"},
		Usage:   "Restore docker container, volumes, ...",
		Action:  RestoreContainer,
	}

	//Define App

	app = &cli.App{
		Name:    "Docker Manager",
		Usage:   "Fully manage your docker environment",
		Version: "0.0.1",
		Flags:   []cli.Flag{&allconfigfilesflag, &configfilesdirflag},
		Commands: []*cli.Command{
			&runcommand,
			&testcommand,
			&backupcommand,
			&restorecommand},
	}
}

func definelogging() {
	file, err := os.OpenFile("/home/chris/bin/info.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}
	log.SetOutput(file)
}

func main() {
	definelogging()
	defineapp()
	exitcode := 0
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println("Fatal error:", err)
		log.Fatal(err)
		exitcode = 555
	}
	os.Exit(exitcode)
}
