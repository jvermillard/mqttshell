# mqttshell

Simple experiment to tunnel shell over MQTT and websocket (don't take this seriously)

The server create a [pty](https://man7.org/linux/man-pages/man7/pty.7.html) attach an instance of `bash` to it.
Then all the output of the `pty` are published over MQTT.

The client connect to the MQTT broker and expose a webpage using [xtermjs](http://xtermjs.org/) connected to the client using websocket and the client forward all the input to the server using MQTT.
