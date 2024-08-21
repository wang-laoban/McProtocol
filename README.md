# Mitsubishi PLC Protocol Library

This Go library provides an implementation of Mitsubishi PLC communication protocols, specifically the A_1E and Qna_3E protocols. The library currently supports reading and writing methods for these protocols. Batch read/write methods and support for the 4E protocol are planned but not yet implemented.

## Features

- **A_1E Protocol:** Implemented methods for reading and writing data.
- **Qna_3E Protocol:** Implemented methods for reading and writing data.
- **Batch Operations:** Planned for future implementation.
- **4E Protocol:** Planned for future implementation.

## Installation

To install this library, you can use the following `go get` command:

```sh
go get -u github.com/wang-laoban/mcprotocol
```

## Usage

Below is an example of how to use this library to connect to a Mitsubishi PLC using the Qna_3E protocol, read a boolean value from a specific memory address, and print the result.

```go
package main

import (
	"fmt"
	mc "github.com/wang-laoban/mcprotocol"
)

func main() {
	// Initialize a new Mitsubishi client for the Qna_3E protocol
	client, err := mc.NewMitsubishiClient(mc.Qna_3E, "127.0.0.1", 6000, 0)
	if err != nil {
		panic(err)
	}

	fmt.Println("Start Connecting")
	err = client.Connect()
	if err != nil {
		panic(err)
	}

	// Read a boolean value from memory address M100
	v, err := client.ReadBool("M100")
	if err != nil {
		panic(err)
	}

	fmt.Println("Read bool:", v)
}
```

## Functions and Methods
### Connecting to PLC

- **NewMitsubishiClient(protocol string, ip string, port int, timeout int):**
 Initializes a new Mitsubishi client with the specified protocol, IP address, port, and timeout.
- **Connect() error:**
Establishes a connection to the PLC.

### Reading and Writing Data

- **ReadBool(address string) (bool, error):**
 Reads a boolean value from the specified memory address.

- **WriteBool(address string, value bool) error:**
Writes a boolean value to the specified memory address.
## Contribution
Contributions to the library are welcome. If you find any issues or have suggestions for improvements, feel free to open an issue or submit a pull request.

## License
This project is licensed under the MIT License. See the LICENSE file for details.