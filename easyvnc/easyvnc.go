package easyvnc

// refer to http://srgia.com/docs/rfbprotocol3.8.html

import (
	"fmt"
	"math"
	"net"
	"strconv"
)

type EasyVNC struct {
	name                      string
	frame_width, frame_height int
	frame                     []int
}

var (
	ch *net.TCPConn = nil
)

func WriteByte(conn *net.TCPConn, err error, value int) error {
	if err != nil {
		return err
	}
	buf := []byte{
		byte(0xff & value),
	}
	_, err = conn.Write(buf)
	return err
}

func WriteShort(conn *net.TCPConn, err error, value int) error {
	if err != nil {
		return err
	}
	buf := []byte{
		byte(0xff & (value >> 8)),
		byte(0xff & (value >> 0)),
	}
	_, err = conn.Write(buf)
	return err
}

func WriteInt(conn *net.TCPConn, err error, value int) error {
	if err != nil {
		return err
	}
	buf := []byte{
		byte(0xff & (value >> 24)),
		byte(0xff & (value >> 16)),
		byte(0xff & (value >> 8)),
		byte(0xff & (value >> 0)),
	}
	_, err = conn.Write(buf)
	return err
}

func ReadByte(conn *net.TCPConn, err error) (int, error) {
	if err != nil {
		return 0, err
	}
	buf := make([]byte, 1)
	size, err := conn.Read(buf)
	if err != nil || size != 1 {
		return 0, err
	}
	value := 0xff & int(buf[0])
	return value, nil
}

func ReadShort(conn *net.TCPConn, err error) (int, error) {
	if err != nil {
		return 0, err
	}
	buf := make([]byte, 2)
	size, err := conn.Read(buf)
	if err != nil || size != 2 {
		return 0, err
	}
	value := (0xff & int(buf[0])) << 8
	value |= (0xff & int(buf[1])) << 0
	return value, nil
}

func ReadInt(conn *net.TCPConn, err error) (int, error) {
	if err != nil {
		return 0, err
	}
	buf := make([]byte, 4)
	size, err := conn.Read(buf)
	if err != nil || size != 4 {
		return 0, err
	}
	value := (0xff & int(buf[0])) << 24
	value |= (0xff & int(buf[1])) << 16
	value |= (0xff & int(buf[2])) << 8
	value |= (0xff & int(buf[3])) << 0
	return value, nil
}

func ReadSkip(conn *net.TCPConn, err error, size int) error {
	if err != nil {
		return err
	}
	buf := make([]byte, size)
	n, err := conn.Read(buf)
	if err != nil || n != size {
		return err
	}
	return nil
}

func sendVncMessage(buf []byte) {
	if ch != nil {
		_, _ = ch.Write(buf)
		// fmt.Printf("sent: %d\n", n)
	}
}

func NewEasyVNC(port int, width, height int) (*EasyVNC, error) {
	laddr, err := net.ResolveTCPAddr("tcp4",
		"0.0.0.0:"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}
	listener, err := net.ListenTCP("tcp4", laddr)
	if err != nil {
		return nil, err
	}
	vnc := EasyVNC{
		name:         "EasyVNC",
		frame_width:  width,
		frame_height: height,
	}
	vnc.frame = make([]int, vnc.frame_width*vnc.frame_height)
	go func() {
		for {
			conn, err := listener.AcceptTCP()
			if err == nil {
				go vnc.doNegotiation(conn)
			}
		}
	}()
	return &vnc, nil
}

func (vnc *EasyVNC) doNegotiation(conn *net.TCPConn) error {
	defer conn.Close()

	buf := make([]byte, 1024)
	_, err := conn.Write([]byte("RFB 003.008\n"))
	if err != nil {
		return err
	}
	_, err = conn.Read(buf) // "RFB 003.008\n"
	if err != nil {
		return err
	}
	fmt.Println("Read RFB")

	_, err = conn.Write([]byte{1, /* Length */
		1 /* Security None */})
	if err != nil {
		return err
	}
	auth, err := ReadByte(conn, nil) // Read auth method
	if err != nil {
		return err
	}
	fmt.Println("Read auth method: " + strconv.Itoa(auth))
	if auth != 1 {
		return nil
	}
	err = WriteInt(conn, nil, 0) // Success
	if err != nil {
		return err
	}

	_, err = ReadByte(conn, nil) // Read shared flag
	if err != nil {
		return err
	}
	fmt.Println("Read shared flag")
	err = WriteShort(conn, nil, vnc.frame_width)  // Frame Width
	err = WriteShort(conn, err, vnc.frame_height) // Frame Height
	err = WriteByte(conn, err, 32)                // Bits Per Pixel
	err = WriteByte(conn, err, 24)                // Depth
	err = WriteByte(conn, err, 0)                 // Big Endian Flag
	err = WriteByte(conn, err, 1)                 // True Color Flag
	err = WriteShort(conn, err, 0xff)             // Red MAX
	err = WriteShort(conn, err, 0xff)             // Green MAX
	err = WriteShort(conn, err, 0xff)             // Blue MAX
	err = WriteByte(conn, err, 16)                // Red Shift
	err = WriteByte(conn, err, 8)                 // Green Shift
	err = WriteByte(conn, err, 0)                 // Blue Shift
	err = WriteByte(conn, err, 0)                 // Padding
	err = WriteByte(conn, err, 0)                 // Padding
	err = WriteByte(conn, err, 0)                 // Padding
	err = WriteInt(conn, err, len(vnc.name))
	if err == nil {
		_, err = conn.Write([]byte(vnc.name))
	}
	if err != nil {
		return err
	}
	ch = conn
	fmt.Println("Start.")
	vnc.sendDummyFrameData()
	vnc.SendAllFrameData()
	return vnc.loop(conn)
}

func (vnc *EasyVNC) loop(conn *net.TCPConn) error {
	for {
		ptype, err := ReadByte(conn, nil)
		if err != nil {
			return err
		}
		switch ptype {
		case 0: // SetPixelFormat
			err := ReadSkip(conn, nil, 19)
			if err != nil {
				return err
			}

		case 2: // SetEncodings
			_, err := ReadByte(conn, nil)
			num, err := ReadShort(conn, err)
			if err != nil {
				return err
			}
			for i := 0; i < num; i++ {
				err = ReadSkip(conn, err, 4)
			}
			if err != nil {
				return err
			}

		case 3: // Request Paint
			_, err := ReadByte(conn, nil) // incremental flag
			x, err := ReadShort(conn, err)
			y, err := ReadShort(conn, err)
			width, err := ReadShort(conn, err)
			height, err := ReadShort(conn, err)
			if err != nil {
				return err
			}
			// fmt.Printf("request (%d, %d) : %d x %d.\n", x, y, width, height)
			// sendFrameData(vnc, x, y, width, height)
			_, _, _, _ = x, y, width, height

		case 5: // mouse event
			mask, err := ReadByte(conn, nil)
			x, err := ReadShort(conn, err)
			y, err := ReadShort(conn, err)
			if err != nil {
				return err
			}
			// fmt.Printf("mouse %d, %d : %d.\n", x, y, mask)
			if mask == 1 {
				// sendFrameData(vnc, 0, 0, vnc.frame_width, vnc.frame_height)
				// sendFrameData(vnc, 0, 0, 300, 200)
				_ = mask
			}
			_, _, _ = x, y, mask

		default:
			fmt.Printf("type: %d\n", ptype)
		}
	}
}

func (vnc *EasyVNC) sendDummyFrameData() {
	x := 0
	y := 0
	width := vnc.frame_width
	height := vnc.frame_height
	// fmt.Printf("frame %d, %d : %d x %d.\n", x, y, width, height)
	if width == 0 || height == 0 {
		return
	}
	buf := make([]byte, 0)
	buf = append(buf, byte(0)) // Type
	buf = append(buf, byte(0)) // Padding
	buf = append(buf, byte(0)) // Len
	buf = append(buf, byte(1))
	buf = append(buf, byte(0xff&(x>>8)))
	buf = append(buf, byte(0xff&(x>>0)))
	buf = append(buf, byte(0xff&(y>>8)))
	buf = append(buf, byte(0xff&(y>>0)))
	buf = append(buf, byte(0xff&(width>>8)))
	buf = append(buf, byte(0xff&(width>>0)))
	buf = append(buf, byte(0xff&(height>>8)))
	buf = append(buf, byte(0xff&(height>>0)))
	buf = append(buf, byte(0)) // Encoding Type
	buf = append(buf, byte(0))
	buf = append(buf, byte(0))
	buf = append(buf, byte(0))
	// fmt.Printf("x: %d\n", x)
	// fmt.Printf("Width: %d\n", width)
	// fmt.Printf("F Width: %d\n", vnc.frame_width)
	buf2 := make([]byte, width*height*4)
	buf = append(buf, buf2...)
	sendVncMessage(buf)
}

func (vnc *EasyVNC) SendFrameData(x, y, width, height int) {
	// fmt.Printf("frame %d, %d : %d x %d.\n", x, y, width, height)
	if width == 0 || height == 0 {
		return
	}
	buf := make([]byte, 0)
	buf = append(buf, byte(0)) // Type
	buf = append(buf, byte(0)) // Padding
	buf = append(buf, byte(0)) // Len
	buf = append(buf, byte(1))
	buf = append(buf, byte(0xff&(x>>8)))
	buf = append(buf, byte(0xff&(x>>0)))
	buf = append(buf, byte(0xff&(y>>8)))
	buf = append(buf, byte(0xff&(y>>0)))
	buf = append(buf, byte(0xff&(width>>8)))
	buf = append(buf, byte(0xff&(width>>0)))
	buf = append(buf, byte(0xff&(height>>8)))
	buf = append(buf, byte(0xff&(height>>0)))
	buf = append(buf, byte(0)) // Encoding Type
	buf = append(buf, byte(0))
	buf = append(buf, byte(0))
	buf = append(buf, byte(0))
	// fmt.Printf("x: %d\n", x)
	// fmt.Printf("Width: %d\n", width)
	// fmt.Printf("F Width: %d\n", vnc.frame_width)
	buf2 := make([]byte, width*height*4)
	for y1 := 0; y1 < height; y1++ {
		for x1 := 0; x1 < width; x1++ {
			color := vnc.PGET(x+x1, y+y1)
			r := 0xff & (color >> 16)
			g := 0xff & (color >> 8)
			b := 0xff & (color >> 0)
			buf2[(y1*width+x1)*4+0] = byte(b)
			buf2[(y1*width+x1)*4+1] = byte(g)
			buf2[(y1*width+x1)*4+2] = byte(r)
			// fmt.Printf("Color: %d %d %d\n", r, g, b)
		}
	}
	buf = append(buf, buf2...)
	sendVncMessage(buf)
}

func (vnc *EasyVNC) SendAllFrameData() {
	vnc.SendFrameData(0, 0, vnc.frame_width, vnc.frame_height)
}

func (vnc *EasyVNC) GetWidth() int {
	return vnc.frame_width
}

func (vnc *EasyVNC) GetHeight() int {
	return vnc.frame_height
}

func (vnc *EasyVNC) PSET(x, y int, color int) {
	position := y*vnc.frame_width + x
	if position < len(vnc.frame) && position >= 0 {
		vnc.frame[position] = color
		// sendFrameData(vnc, x, y, 1, 1)
	}
}

func (vnc *EasyVNC) PGET(x, y int) int {
	position := y*vnc.frame_width + x
	color := 0
	if position < len(vnc.frame) && position >= 0 {
		color = vnc.frame[position]
	}
	return color
}

func (vnc *EasyVNC) Line(x1, y1, x2, y2 int, color int) {
	if x1 > x2 {
		tmp := x1
		x1 = x2
		x2 = tmp
	}
	if y1 > y2 {
		tmp := y1
		y1 = y2
		y2 = tmp
	}
	if x2 != x1 && x2-x1 >= y2-y1 {
		for x := x1; x <= x2; x++ {
			y := (x-x1)*(y2-y1)/(x2-x1) + y1
			vnc.PSET(x, y, color)
		}
	} else if y2 != y1 {
		for y := y1; y <= y2; y++ {
			x := (y-y1)*(x2-x1)/(y2-y1) + x1
			vnc.PSET(x, y, color)
		}
	} else {
		vnc.PSET(x1, y1, color)
	}
}

func (vnc *EasyVNC) Arc(x, y int, rx, ry int, color int) {
	if rx == 0 || ry == 0 {
		return
	}
	startdx := int(float64(rx*rx) / math.Sqrt(float64(rx*rx+ry*ry)))
	for dx := startdx; dx >= 0; dx-- {
		dy := int(math.Sqrt(float64(rx*rx-dx*dx))*float64(ry)/float64(rx) + 0.5)
		vnc.PSET(x+dx, y+dy, color)
		vnc.PSET(x+dx, y-dy, color)
		vnc.PSET(x-dx, y+dy, color)
		vnc.PSET(x-dx, y-dy, color)
	}
	startdy := int(float64(ry*ry) / math.Sqrt(float64(rx*rx+ry*ry)))
	for dy := startdy; dy >= 0; dy-- {
		dx := int(math.Sqrt(float64(ry*ry-dy*dy))*float64(rx)/float64(ry) + 0.5)
		vnc.PSET(x+dx, y+dy, color)
		vnc.PSET(x+dx, y-dy, color)
		vnc.PSET(x-dx, y+dy, color)
		vnc.PSET(x-dx, y-dy, color)
	}
}
