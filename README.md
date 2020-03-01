# FHEM wolfmqttbridge based on https://github.com/kgbvax/wolfmqttbridge

WOLF Smartset MQTT Bridge (for FHEM)

## Install

First you need to install go:
`sudo apt-get install golang`

Copy the git repository:
git clone `https://github.com/ste-ta/fhemwolfmqttbridge`

Build the project:

`go build`

Copy the file to /opt/wolfsmartset:

`sudo mkdir /opt/wolfsmartset`

`sudo chmod 755 wolfmqttbridge`  

`sudo cp wolfmqttbridge /opt/wolfsmartset`

Customize init file:

...


It periodically fetches current state information 
from https://www.wolf-smartset.com and publishes this to MQTT - in a way that works with https://www.home-assistant.io.



This works with my Wolf CGS-2L and a Wolflink Pro, everything else _may_ work or not.

Update rate defaults to 20 seconds (which I hope is acceptable since the Wolf-Smartset web-clients polls data every 10 seconds)
## What works
* Talk to Wolf-Smartset.com portal (re-engineered API, if there is a spec for this I would be interested)
* Emit auto-confguration MQTT messages for home-assistant

## What does not work
* Only one device supported (it takes the first device found in the portal)
* No direct connect to bridge in the local network - I could not find a spec for this interface
* This is currently read-only

# Running
For running this on the command-line try --help-long

To support running in a bare container, most args can be passed in as Environment variables, the following variables are mandatory:

* WOLF_USER  - your userid at  https://www.wolf-smartset.com
* WOLF_PW - password for your user at https://www.wolf-smartset.com
* BROKER - address of the MQTT Broker to use, if you are running this as container under hass.io and use the Mosquitto broker add-on this is tcp://core-mosquitto:1883 
* BROKER_USER - username for the MQTT broker (when using hass.io mosquitto a valid hass.io user works)
* BROKER_PW - password for the MQTT broker ( " " )

 
To run this as container on hass.io, use e.g. the Portainer add-on and configure a new container:
* Image: kgbvax/wolfmqttbridge:latest
* ENV: Define the variables listed abover
* Network: Add this to the "hassio" network
* Restart Policy: On Failure / 5 (recommended)
* Resources: As you like should work with 64MB and some tiny CPU

## MQTT Topics
* Topics for values are auto-generated like this: 
   ```wolf/<Value-Name>/state```
    The root topic can be overwritten using WOLF_MQTT_ROOT_TOPIC environment or --rootTopic. Value-Name is the value as it appears on the GUI, (with spaces removed).  Payload is the raw value (as string)
*  Default topic for home-assistant MQTT discovery is ```homeassistant``` (which is HA's default). This can be changed with HA_DISCO_TOPIC or --haDiscoTopic
