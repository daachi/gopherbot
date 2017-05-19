This article deals mainly with writing plugins in one of the scripting languages supported by Gopherbot,
the most popular means for writing new command plugins. For writing native compiled-in plugins in Go, see
`gopherbot/main.go` and the sample plugins in `goplugins/`. API documentation for Robot methods is available
at:

https://godoc.org/github.com/uva-its/gopherbot/bot#Robot

Note that the script plugin API is implemented on top of the native Go API, so that document may also be of use for scripting plugin authors.

Table of Contents
=================

  * [Default Configuration](#default-configuration)
  * [Calling Convention](#calling-convention)
  * [Plugin Types and Calling Events](#plugin-types-and-calling-events)
    * [Command Plugins](#command-plugins)
    * [Authorization Plugins](#authorization-plugins)
    * [Elevation Plugins](#elevation-plugins)
    * [Other Reserved Commands](#other-reserved-commands)
  * [Getting Started](#getting-started)
    * [Starting from a Sample Plugin](#starting-from-a-sample-plugin)
    * [Using Boilerplate Code](#using-boilerplate-code)
      * [Bash Boilerplate](#bash-boilerplate)
      * [PowerShell Boilerplate](#powershell-boilerplate)
      * [Python Boilerplate](#python-boilerplate)
      * [Ruby Boilerplate](#ruby-boilerplate)
  * [The Plugin API](#the-plugin-api)

# Default Configuration
Plugin configuration is fully documented in the [configuration](Configuration.md) article; you should be familiar with that document before beginning to write your own plugins.

On start-up and during a reload, the robot will run each external script plugin with an argument of `configure`. The plugin should respond by writing the plugin default configuration to standard out and exiting with exit code 0. When responding to `configure`, the plugin shouldn't initialize a robot object or make any API calls, as `configure` is called without setting robot environment variables.

# Calling Convention
The robot calls external plugins by creating a goroutine and exec'ing the external script with a set of environment variables. The external script uses the appropriate library for the scripting language to create a robot object from the environment. The script then examines it's command-line arguments to determine the type of action to take (normally a command followed by arguments to the command), and uses the library to make JSON-over-http calls for executing and returning results from methods. Depending on how the plugin is used, different kinds of events can cause external plugins to be called with a variety of commands and arguments. The most common means of calling an external plugin is for one of it's commands to be matched, or by matching a pattern in an ambient message (one not specifically directed to the robot).

# Plugin Types and Calling Events

There are (currently) three different kinds of external plugin:
 * Command / Message Plugins - these are called by the robot in respond to messages the user sends
 * Authorization Plugins - these plugins encapsulate the logic for authorizing specific users to use specific commands, and are called by the robot during authorization processing
 * Elevation Plugins - these plugins perform some variety of multi-factor authentication for higher assurance of user identity, and are called by the robot during elevation processing

In addition, external plugins can call each other using the CallPlugin(plugin, command, args...) method, subject to the target plugin's `TrustedPlugins` and `TrustAllPlugins` settings.

## Command Plugins

A command plugin's configuration specifies `CommandMatchers` and `MessageMatchers` that associate regular expressions with plugin commands:
```yaml
MessageMatchers:
- Command: help
  Regex: '^(?i:help)$'
CommandMatchers:
- Regex: (?i:remember ([-\w .,!?:\/]+))
  Command: remember
  Contexts: [ "item" ]
```
Whenever a `CommandMatcher` regex matches a command given to the robot, or a `MessageMatcher` matches an ambient message, the robot calls the plugin script with the first argument being the matched `Command`, and subsequent arguments corresponding to the regex capture groups (which may in some cases be an empty string). Command plugins should normally exit with status 0 (bot.Succeed), or non-zero for unusual error conditions that may require an administrator to investigate. The robot will notify the user whenever a command plugin exits non-zero, or when it emits output to STDERR.

## Authorization Plugins
To separate command logic from user authorization logic, Gopherbot supports the concept of an **authorization plugin**. The main `gopherbot.yaml` can define a specific plugin as the `DefaultAuthorizer`, and individual plugins can be configured to override this value by specifying their own `Authorizer` plugin. If a plugin lists any commands in it's `AuthorizedCommands` config item, or specifies `AuthorizeAllCommands: true`, then the robot will call the authorizer plugin with a command of `authorize`, followed by four arguments:
 * The name of the plugin for which authorization is being requested
 * The plugin command being called
 * The name of the calling plugin, if called via CallPlugin(...)
 * The optional value of `AuthRequire`, which may be interpreted as a group or role

Based on these values and the `User` and `Channel` values from the robot, the authorization plugin should evaluate whether a user/plugin is authorized for the given command and exit with one of:
 * bot.Succeed (0) - authorized
 * bot.Fail (1) - not authorized
 * bot.MechanismFail (2) - a technical issue prevented the robot from determining authorization

Additionally, authorization plugins should provide feedback to the user on `Fail` or `MechanismFail` so they can have the issue addressed, e.g. "Authorization failed: user not a member of group 'foo'"

## Elevation Plugins
Elevation plugins provide the means to request additional authentication from the user for commands where higher assurance of identity is desired. The main `gopherbot.yaml` can specify an elevation plugin as the `DefaultElevator`, which can be overridden by a given plugin specifying an `Elevator`. When the plugin lists commands as `ElevatedCommands` or `ElevateImmediateCommands`, the robot will call the appropriate elevator plugin with a command of `elevate` and a first argument of `true` or `false` for `immediate`. The elevator plugin should interpret `immediate == true` to mean MFA is required every time; when `immediate != true`, successful elevation may persist for a configured timeout period.

Based on the result of the elevation determination, the plugin should have an exit status one of:
 * bot.Succeed (0) - elevation succeeded
 * bot.Fail (1) - elevation failed
 * bot.MechanismFail (2) - a technical issue prevented the robot from processing the elevation request

Additionally, the elevation plugin should provide feedback to the user when elevation isn't successful to indicate the nature of the failure.

## Other Reserved Commands
In addition to the `configure` command, which instructs a plugin to dump it's default configuration to standard out, the following commands are reserved:
* `init` - During startup and reload, the robot will call external plugins with a command argument of `init`. Since all environment variables for the robot are set at that point, it would be possible to e.g. save a robot data structure that could be loaded and used in a cron job.
* `event` - This command is reserved for future use with e.g. user presence change & channel join/leave events

# Getting Started
## Starting from a Sample Plugin
The simplest way for a new plugin author to get started is to:
* Disable the demo plugin for your chosen scripting language (if enabled) in `<config dir>/conf/gopherbot.yaml`
* Copy the demo plugin to `<config dir>/plugins/<newname>(.extension)`
* Enable your new plugin in `gopherbot.yaml` and give it a descriptive `Name`

## Using Boilerplate Code
Each supported scripting language has a certain amount of "boilerplate" code required in every command plugin; generally, the boilerplate code is responsible for:
* Loading the appropriate version of the Gopherbot library from `$GOPHER_INSTALLDIR/lib`
* Defining and providing the default config
* Instantiating a Robot object with a library call
Normally this is followed by some form of case / switch statement that performs different functions based on the contents of the first argument, a.k.a. the "command".

### Bash Boilerplate
```bash
#!/bin/bash -e

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

COMMAND=$1
shift

configure(){
  cat <<"EOF"
<yaml config document>
EOF
}

case "$COMMAND" in
	"configure")
		configure
		;;
...
```
**NOTE:** Bash doesn't have an object-oriented API

### PowerShell Boilerplate
```powershell
#!powershell.exe
# -or-
#!C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe

# Stylistic, can be omitted; $cmdArgs is always a String[],
# but $Args turns into a String when you shift off the 2nd item
[String[]]$cmdArgs = $Args
Import-Module "$Env:GOPHER_INSTALLDIR\lib\gopherbot_v1.psm1"
$bot = Get-Robot

$config = @'
<yaml config document>
'@

$command, $cmdArgs = $cmdArgs

switch ($command)
{
  "configure" {
    Write-Output $config
    exit
  }
 ...
}
```

### Python Boilerplate
```python
#!/usr/bin/python

import os
import sys
sys.path.append("%s/lib" % os.getenv("GOPHER_INSTALLDIR"))
from gopherbot_v1 import Robot

bot = Robot()

default_config = '''
<yaml config document>
'''

executable = sys.argv.pop(0)
command = sys.argv.pop(0)

if command == "configure":
    print default_config
...
```
### Ruby Boilerplate
```ruby
#!/usr/bin/ruby

# boilerplate
require ENV["GOPHER_INSTALLDIR"] + '/lib/gopherbot_v1'

bot = Robot.new()

defaultConfig = <<'DEFCONFIG'
<yaml config document>
DEFCONFIG

command = ARGV.shift()

case command
when "configure"
	puts defaultConfig
	exit
...
end
```
# The Plugin API

Gopherbot has a rich set of methods (functions) for interacting with the robot / user. Here we break down the API into sets of related functions:
* [Message Sending Methods](Message-Sending-API.md) - for sending messages to the users
* [Attribute Retrieval Methods](Attribute-Retrieval-API.md) - for retrieving names, email addresses, etc.
* [Response Request Methods](Reponse-Request-API.md) - for getting replies from the user
* [Long-term Memory Methods](Long-term-Memory-API.md) - for storing long-term memories (like a TODO list, or user preference)
* [Short-term Memory Methods](Short-term-Memory-API.md) - for storing short-term memories like conversation context that are stored in memory and expire after a period of time
* [Security and Elevation Methods](Security-API.md) - for making determinations on privileged commands
* [Utility Methods](Utility-API.md) - a collection of miscellaneous useful functions, like Pause()
