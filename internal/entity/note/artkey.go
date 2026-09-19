package note

type ArtKey string

var (
	Artp1    ArtKey = "p1"
	Artp2    ArtKey = "p2"
	Artp3    ArtKey = "p3"
	Artx1    ArtKey = "x1"
	Artx2    ArtKey = "x2"
	Artx3    ArtKey = "x3"
	Artx7    ArtKey = "x7"
	Artx8    ArtKey = "x8"
	Artx9    ArtKey = "x9"
	Artsp1   ArtKey = "sp1"
	Artx10_1 ArtKey = "x10_1"
	Artx10_2 ArtKey = "x10_2"
	Artx10_3 ArtKey = "x10_3"
	Artx10_6 ArtKey = "x10_6"
	Artx10_4 ArtKey = "x10_4"
	Artx10_5 ArtKey = "x10_5"
	Artx11_1 ArtKey = "x11_1"
	Artx11_2 ArtKey = "x11_2"
	Artx11_3 ArtKey = "x11_3"
	Artx11_4 ArtKey = "x11_4"
	Artx6    ArtKey = "x6"
	Artx4    ArtKey = "x4"
	Artx5    ArtKey = "x5"
	Arte1    ArtKey = "e1"
	Arte2    ArtKey = "e2"
	Arte3    ArtKey = "e3"
	Arte4    ArtKey = "e4"
	Arte5    ArtKey = "e5"
	Arte6    ArtKey = "e6"
	Arte7    ArtKey = "e7"
	Artn1    ArtKey = "n1"
	Artn2    ArtKey = "n2"
	Artn3    ArtKey = "n3"
	Artn4    ArtKey = "n4"
	Artn5    ArtKey = "n5"
	Artn6    ArtKey = "n6"
	Artn7    ArtKey = "n7"
	Artn8    ArtKey = "n8"
)

var validArtKeys = map[ArtKey]struct{}{}

func init() {
	for _, key := range []ArtKey{
		Artp1, Artp2, Artp3, Artx1, Artx2, Artx3, Artx7, Artx8, Artx9, Artsp1,
		Artx10_1, Artx10_2, Artx10_3, Artx10_6, Artx10_4, Artx10_5,
		Artx11_1, Artx11_2, Artx11_3, Artx11_4, Artx6, Artx4, Artx5,
		Arte1, Arte2, Arte3, Arte4, Arte5, Arte6, Arte7,
		Artn1, Artn2, Artn3, Artn4, Artn5, Artn6, Artn7, Artn8,
	} {
		validArtKeys[key] = struct{}{}
	}
}

func (key ArtKey) Valid() bool {
	_, ok := validArtKeys[key]
	return ok
}
