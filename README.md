# WOLF Smartset MQTT Bridge (for FHEM https://fhem.de/) based on https://github.com/kgbvax/wolfmqttbridge

I was looking for a way to integrate my Wolf heating with FHEM and came along the wolfmqttbridge. I added a init file to make it run as a service and added a describtion how to integrate with FHEM.

## Install
Prerequisite: You have a local MQTT server running.

### First you need to install go:
`sudo apt-get install golang`

### Copy the git repository:
`git clone https://github.com/ste-ta/fhemwolfmqttbridge`

### Build the project:

`go build`

### Copy the executable to /opt/wolfsmartset:

`sudo mkdir /opt/wolfsmartset`

`sudo chmod 755 wolfmqttbridge`  

`sudo cp wolfmqttbridge /opt/wolfsmartset`

### Install service and start:

Change username and passwort and MQTT-Brokersettings in wolfsmartset.init

Copy wolfsmartset.init to init.d

`sudo cp wolfsmartset.init /etc/init.d/wolfsmartset`

Start the service:

`sudo /etc/init.d/wolfsmartset start`


### MQTT

wolfmqttbridge will create a topic called wolf MQTT with various parameters

## FHEM integration

In order to integrate into FHEM you will need MQTT https://wiki.fhem.de/wiki/MQTT:

### Install MQTT module

 `sudo cpan install Net::MQTT:Simple`
 
 `sudo cpan install Net::MQTT:Constants`

Create MQTT connection:

`define mqtt MQTT 127.0.0.1:1883`

Create Wolf MQTT device:
	
`define mywolf MQTT_DEVICE`

Set autocreate to wolftopic:

`attr mywolf autoSubscribeReadings wolf/+/state`

# Info from original readme

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
