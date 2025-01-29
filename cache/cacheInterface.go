package cache

type ObjectMetadata struct {
	Host   string
	Bucket string
	Key    string
}

type Object struct {
	Data     *[]byte
	Metadata *ObjectMetadata
}

type Initializer func() (*Object, error)

type Cache interface {
	Get(key string, initializer Initializer) (*Object, error)
	GetMetadata(key string) (*ObjectMetadata, error)
	Put(*Object) error
}
