# SocketCAN User Interface for the Terminal - socanui

<p align="center">
<img src="media/ubuntu.png" width="1024" alt="socanui" title="socanui" />
</p>


## Features

- Send and Receive CAN Message
- CAN Frame List
- CAN Frame Table
- Show CAN Interface Parameter
- CAN Statistics
- Send CAN Frames (single, repeated, random)
- Filter CAN Frames
  
## Usage

Just run `socanui <interface>` in your terminal and the UI will start.

For the first physical CAN adapter:
```sh
socanui can0
```

## Install

```sh
git clone https://github.com/miwagner/socanui.git
go build -o socanui main.go
```

## Socket CAN

You can create a virtual CAN interface if you don't have a physical CAN adapter:
```sh
sudo ip link add dev vcan0 type vcan
sudo ip link set up vcan0
```


You can generate testdata as follow:
```sh
cangen vcan0
```