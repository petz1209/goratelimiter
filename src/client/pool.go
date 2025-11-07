package client

func NewPool(Addr string, Size int) (*Pool, error) {

	f := func() *RateLimitClient {
		return NewRateLimitClient(Addr)
	}

	p := &Pool{Size: Size, connections: make(chan *RateLimitClient, Size+1), factory: f}

	for range p.Size {
		p.connections <- p.factory()
	}
	return p, nil

}

type Pool struct {
	Size        int
	connections chan *RateLimitClient
	factory     func() *RateLimitClient
}

func (p *Pool) Close() {

	for conn := range p.connections {
		conn.Close()
	}

}

func (p *Pool) GetConnection() *RateLimitClient {
	// fmt.Println("wait for connection from pool")
	c := <-p.connections
	// fmt.Println("aquired connection from pool")
	return c
}

func (p *Pool) ReleaseConnection(conn *RateLimitClient) {
	p.connections <- conn
	// fmt.Println("connection returned to pool")
}

func (p *Pool) Aquire(key string, maxConcurrency, maxVolume int) (int, error) {
	conn := p.GetConnection()
	defer p.ReleaseConnection(conn)
	return conn.Aquire(key, maxConcurrency, maxVolume)
}

func (p *Pool) Return(key string, volume int) (int, error) {
	conn := p.GetConnection()
	defer p.ReleaseConnection(conn)
	return conn.Return(key, volume)
}
