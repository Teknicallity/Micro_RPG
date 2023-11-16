package main

import (
	"embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/lafriks/go-tiled"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
	"log"
	"path"
)

//go:embed assets/*
var EmbeddedAssets embed.FS

const (
	FRAMES_PER_ROW = 4
	FRAME_COUNT    = 8
)

const (
	DOWN = iota
	RIGHT
	UP
	LEFT
)

const (
	WALK = iota
	ATTACK
)

const (
	CHARACTRIGHT = iota
	CHARACTLEFT
)

type rpgGame struct {
	level    *tiled.Map
	tileHash map[uint32]*ebiten.Image
	player   character
	chs      []character
}

type character struct {
	spriteSheet  *ebiten.Image
	xLoc         int
	yLoc         int
	inventory    []item
	direction    int
	frame        int
	frameDelay   int
	FRAME_HEIGHT int
	FRAME_WIDTH  int
	action       int
	imageYOffset int
}

type item struct {
	picture     *ebiten.Image
	displayName string
}

func (game *rpgGame) Update() error {
	getPlayerInput(game)

	animatePlayerSprite(&game.player)
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		game.player.action = ATTACK
	} else {
		game.player.action = WALK
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) && game.player.action == WALK {
		game.player.xLoc -= 3
	} else if ebiten.IsKeyPressed(ebiten.KeyD) && game.player.action == WALK {
		game.player.xLoc += 3
	} else if ebiten.IsKeyPressed(ebiten.KeyW) && game.player.action == WALK {
		game.player.yLoc -= 3
	} else if ebiten.IsKeyPressed(ebiten.KeyS) && game.player.action == WALK {
		game.player.yLoc += 3
	}

	for i := range game.chs {
		if game.chs[i].action == WALK {
			game.chs[i].frameDelay += 1
			if game.chs[i].frameDelay%8 == 0 {
				game.chs[i].frame += 1
				if game.chs[i].frame >= 4 {
					game.chs[i].frame = 0
				}
			}
		}
	}

	return nil
}

func (game *rpgGame) Draw(screen *ebiten.Image) {
	//screen.Fill(colornames.Blue)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Reset()
	///*
	for tileY := 0; tileY < game.level.Height; tileY++ {
		for tileX := 0; tileX < game.level.Width; tileX++ {
			op.GeoM.Reset()
			//get on screen position
			tileXPos := float64(game.level.TileWidth * tileX)
			tileYPos := float64(game.level.TileHeight * tileY)
			op.GeoM.Translate(tileXPos, tileYPos)

			// Get the tile ID from the appropriate LAYER
			tileToDraw := game.level.Layers[0].Tiles[tileY*game.level.Width+tileX]

			// Retrieve the corresponding sub-image from the map
			ebitenTileToDraw, ok := game.tileHash[tileToDraw.ID]
			if !ok {
				// Handle the case where the tile ID is not found in the map
				fmt.Printf("Tile ID %d not found in tileHash\n", tileToDraw.ID)
				continue
			}

			// Draw the sub-image
			screen.DrawImage(ebitenTileToDraw, op)
		}
	} //*/

	drawPlayerFromSpriteSheet(op, screen, game.player)
	for i := range game.chs {
		drawImageFromSpriteSheet(op, screen, game.chs[i])
	}
	//DrawCenteredText(screen, font.Face("Comic Sans"), "hello", 200, 100)
	img := ebiten.NewImage(300, 100)
	addLabel(img, 20, 30, "Hello Go")
	op.GeoM.Reset()
	screen.DrawImage(img, op)
}

func drawPlayerFromSpriteSheet(op *ebiten.DrawImageOptions, screen *ebiten.Image, targetCharacter character) {
	op.GeoM.Reset()
	op.GeoM.Scale(3, 3)
	op.GeoM.Translate(float64(targetCharacter.xLoc), float64(targetCharacter.yLoc))
	screen.DrawImage(targetCharacter.spriteSheet.SubImage(
		image.Rect(
			targetCharacter.frame*targetCharacter.FRAME_WIDTH,
			targetCharacter.direction*targetCharacter.FRAME_HEIGHT,
			targetCharacter.frame*targetCharacter.FRAME_WIDTH+targetCharacter.FRAME_WIDTH,
			targetCharacter.direction*targetCharacter.FRAME_HEIGHT+targetCharacter.FRAME_HEIGHT)).(*ebiten.Image), op)
}

func drawImageFromSpriteSheet(op *ebiten.DrawImageOptions, screen *ebiten.Image, targetCharacter character) {
	op.GeoM.Reset()
	resize := float64(3)
	if targetCharacter.direction == CHARACTLEFT {
		op.GeoM.Scale(resize, resize)
		op.GeoM.Translate(float64(targetCharacter.xLoc), float64(targetCharacter.yLoc))
	} else if targetCharacter.direction == CHARACTRIGHT {
		op.GeoM.Scale(-resize, resize)
		op.GeoM.Translate(
			float64(targetCharacter.xLoc)+(float64(targetCharacter.FRAME_WIDTH)*resize), float64(targetCharacter.yLoc))
	}
	screen.DrawImage(targetCharacter.spriteSheet.SubImage(
		image.Rect(
			targetCharacter.frame*targetCharacter.FRAME_WIDTH,
			targetCharacter.imageYOffset*targetCharacter.FRAME_HEIGHT,
			targetCharacter.frame*targetCharacter.FRAME_WIDTH+targetCharacter.FRAME_WIDTH,
			targetCharacter.FRAME_HEIGHT+targetCharacter.FRAME_HEIGHT*targetCharacter.imageYOffset)).(*ebiten.Image), op)
}

func animatePlayerSprite(targetCharacter *character) {
	if targetCharacter.action == WALK {
		targetCharacter.frameDelay += 1
		if targetCharacter.frameDelay%8 == 0 {
			targetCharacter.frame += 1
			if targetCharacter.frame >= 4 {
				targetCharacter.frame = 0
			}
		}
	} else if targetCharacter.action == ATTACK {
		if 4 <= targetCharacter.frame && targetCharacter.frame <= 7 {
			targetCharacter.frameDelay += 1
			if targetCharacter.frameDelay%8 == 0 {
				targetCharacter.frame--
				if targetCharacter.frame <= 4 {
					targetCharacter.frame = 7
				}
			}
		} else {
			targetCharacter.frame = 7
		}
	}
}

func (game rpgGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight //by default, just return the current dimensions
}

func main() {
	ebiten.SetWindowTitle("SimpleRPG")

	gameMap := loadMapFromEmbedded(path.Join("assets", "world.tmx"))
	ebiten.SetWindowSize(gameMap.TileWidth*gameMap.Width, gameMap.TileHeight*gameMap.Height)
	ebitenImageMap := makeEbitenImagesFromMap(*gameMap)

	playerSpriteSheet := LoadEmbeddedImage("characters", "player.png")
	enemySpriteSheet := LoadEmbeddedImage("characters", "characters.png")

	user := character{
		spriteSheet:  playerSpriteSheet,
		xLoc:         250,
		yLoc:         250,
		direction:    RIGHT,
		frame:        0,
		frameDelay:   0,
		FRAME_HEIGHT: 32,
		FRAME_WIDTH:  16,
		imageYOffset: -1,
	}

	mannequin := character{
		spriteSheet:  enemySpriteSheet,
		xLoc:         100,
		yLoc:         100,
		inventory:    nil,
		direction:    CHARACTLEFT,
		frame:        0,
		frameDelay:   0,
		FRAME_HEIGHT: 32,
		FRAME_WIDTH:  32,
		action:       WALK,
		imageYOffset: 0,
	}

	king := character{
		spriteSheet:  enemySpriteSheet,
		xLoc:         100,
		yLoc:         200,
		inventory:    nil,
		direction:    CHARACTLEFT,
		frame:        0,
		frameDelay:   0,
		FRAME_HEIGHT: 32,
		FRAME_WIDTH:  32,
		action:       WALK,
		imageYOffset: 1,
	}

	leprechaun := character{
		spriteSheet:  enemySpriteSheet,
		xLoc:         300,
		yLoc:         300,
		inventory:    nil,
		direction:    CHARACTRIGHT,
		frame:        0,
		frameDelay:   0,
		FRAME_HEIGHT: 32,
		FRAME_WIDTH:  32,
		action:       WALK,
		imageYOffset: 2,
	}
	enemies := make([]character, 0, 5)
	enemies = append(enemies, mannequin)
	enemies = append(enemies, king)
	enemies = append(enemies, leprechaun)

	game := rpgGame{
		level:    gameMap,
		tileHash: ebitenImageMap,
		player:   user,
		chs:      enemies,
	}
	err := ebiten.RunGame(&game)
	if err != nil {
		fmt.Println("Failed to run game", err)
	}
}

func LoadEmbeddedImage(folderName string, imageName string) *ebiten.Image {
	embeddedFile, err := EmbeddedAssets.Open(path.Join("assets", folderName, imageName))
	if err != nil {
		log.Fatal("failed to load embedded image ", imageName, err)
	}
	ebitenImage, _, err := ebitenutil.NewImageFromReader(embeddedFile)
	if err != nil {
		fmt.Println("Error loading tile image:", imageName, err)
	}
	return ebitenImage
}

func loadMapFromEmbedded(name string) *tiled.Map {
	embeddedMap, err := tiled.LoadFile(name, tiled.WithFileSystem(EmbeddedAssets))
	if err != nil {
		fmt.Println("Error loading embedded map:", err)
	}
	return embeddedMap
}

//func makeEbitenImagesFromMap(gameMap tiled.Map) map[uint32]*ebiten.Image {
//	idToImage := make(map[uint32]*ebiten.Image)
//	for _, tile := range gameMap.Tilesets[0].Tiles {
//		embeddedFile, err := EmbeddedAssets.Open(path.Join("assets", tile.Image.Source))
//		if err != nil {
//			log.Fatal("failed to load embedded image ", embeddedFile, err)
//		}
//		ebitenImageTile, _, err := ebitenutil.NewImageFromReader(embeddedFile)
//		if err != nil {
//			fmt.Println("Error loading tile image:", tile.Image.Source, err)
//		}
//		idToImage[tile.ID] = ebitenImageTile
//	}
//	return idToImage
//}

func makeEbitenImagesFromMap(tiledMap tiled.Map) map[uint32]*ebiten.Image {
	idToImage := make(map[uint32]*ebiten.Image)
	tilesetImagePath := path.Join("assets", tiledMap.Tilesets[0].Image.Source)
	embeddedFile, err := EmbeddedAssets.Open(tilesetImagePath)
	if err != nil {
		log.Fatal("failed to load embedded image ", tilesetImagePath, err)
	}
	ebitenImageTileset, _, err := ebitenutil.NewImageFromReader(embeddedFile)
	if err != nil {
		fmt.Println("Error loading tileset image:", tilesetImagePath, err)
	}

	// Create sub-images for each tile and store them in the map
	for _, tile := range tiledMap.Tilesets[0].Tiles {
		x := int(tile.Image.Width) * (int(tile.ID) % int(tiledMap.Tilesets[0].Columns))
		y := int(tile.Image.Height) * (int(tile.ID) / int(tiledMap.Tilesets[0].Columns))

		subImageRect := image.Rect(x, y, x+int(tile.Image.Width), y+int(tile.Image.Height))
		subImage := ebitenImageTileset.SubImage(subImageRect).(*ebiten.Image)

		fmt.Println(tile.ID)
		idToImage[tile.ID] = subImage
	}

	return idToImage
}

func getPlayerInput(game *rpgGame) {
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		game.player.direction = LEFT
	} else if ebiten.IsKeyPressed(ebiten.KeyD) {
		game.player.direction = RIGHT
	} else if ebiten.IsKeyPressed(ebiten.KeyW) {
		game.player.direction = UP
	} else if ebiten.IsKeyPressed(ebiten.KeyS) {
		game.player.direction = DOWN
	}
}

func DrawCenteredText(screen *ebiten.Image, font font.Face, s string, cx, cy int) { //from https://github.com/sedyh/ebitengine-cheatsheet
	bounds := text.BoundString(font, s)
	x, y := cx-bounds.Min.X-bounds.Dx()/2, cy-bounds.Min.Y-bounds.Dy()/2
	text.Draw(screen, s, font, x, y, colornames.White)
}

func addLabel(img *ebiten.Image, x, y int, label string) {
	// from https://stackoverflow.com/a/38300583
	col := color.RGBA{100, 200, 0, 255}
	point := fixed.Point26_6{fixed.I(x), fixed.I(y)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}
