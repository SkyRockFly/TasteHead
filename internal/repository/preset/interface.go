package preset

type Profile struct {
	Name      string
	URL       string
	Selectors Selectors
}

type Selectors struct {
	Post      string
	ImageAttr string
	NextPage  string
}

type Repository interface {
	Create(profile Profile) error
	Delete(name string) error
	List() []Profile
}
