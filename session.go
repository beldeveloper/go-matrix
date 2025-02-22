package gomatrix

type Session struct {
	AccessToken string `json:"access_token"`
	DeviceID    string `json:"device_id"`
}

type SessionStorage interface {
	Set(session Session) error
	Get() (Session, error)
}

type InMemorySessionStorage struct {
	session Session
}

func NewInMemorySessionStorage() *InMemorySessionStorage {
	return &InMemorySessionStorage{}
}

func (s *InMemorySessionStorage) Set(session Session) error {
	s.session = session
	return nil
}

func (s *InMemorySessionStorage) Get() (Session, error) {
	return s.session, nil
}
