# SimpleCLI
A super bare bones cli app for golang that allows for logging and intuitively registering commands.

## Usage:
  First and foremost it's important to note the module registers its own default help & stop commands that cannot be overwritten unless you flag OverwriteCommands in the NewCommandCfg function. 
  If you want to make your own custom stopping command make sure to end it with cfg.running = false to tell the cli to exit
  
  After importing the module run ```NewCommandCfg()``` and save the returned cfg. 
  To register new commands run ```cfg.RegisterCommand(cmdName, func)``` with the cmdName being the string inputted into stdin to call the command. 
  Lastly, run ```cfg.StartConsole()``` which will automatically lock the registering of commands and start up a thread that will listen for commands in stdin. 
  To stop the console either type stop in stdin, or set ```cfg.running = false```. 

---
I kinda rushed to push this onto github cause I got tired of copying the same folder into all my go projects :p
If I get any surge of inspiration I may update this repo but chances are low at best

Thank you for listening :3
