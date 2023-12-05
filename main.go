package main

import (
	"embed"
	"fmt"
	"github.com/co0p/tankism/lib/collision"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/lafriks/go-tiled"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"image"
	"log"
	"math"
	"path"
	"slices"
	"strconv"
)

//go:embed assets/*
var EmbeddedAssets embed.FS

const (
	resizeScale     = 3
	worldScale      = 3
	COOLDOWN        = 60
	soundSampleRate = 48000
)

const (
	DOWN = iota
	RIGHT
	UP
	LEFT
)

const (
	WALK = iota
	INTERACT
	PATH
	//STAY
)

const (
	CHARACTRIGHT = iota
	CHARACTLEFT
)

const (
	NOTTALKED = iota
	TALKED
	RETURNEDITEM
)

type rpgGame struct {
	worldinfo

	barrierRect     []image.Rectangle
	teleporterRects map[uint32]image.Rectangle
	windowWidth     int
	windowHeight    int
	barrierIDs      []uint32
	player          player
	enemies         []character
	questGiver      character
	fontLarge       font.Face
	fontSmall       font.Face
	heartImage      image.Image
	droppedItems    []item
	sounds          sounds
}

type sounds struct {
	enemyDeath     sound
	enemyHit       sound
	attackPowerUp  sound
	heal           sound
	playerInteract sound
	playerDamaged  sound
	questGiverTalk sound
	itemPickup     sound

	audioContext *audio.Context
}

type sound struct {
	audioPlayer *audio.Player
}

func (sound *sound) playSound() {
	//sound.
	err := sound.audioPlayer.Rewind()
	if err != nil {
		fmt.Println("Error rewinding sound: ", err)
	}
	sound.audioPlayer.Play()
}

func loadEmbeddedWavToSound(name string, context *audio.Context) sound {
	file, err := EmbeddedAssets.Open(path.Join("assets", "sounds", name))
	if err != nil {
		fmt.Println("Error Loading embedded sound: ", err)
	}
	soundWav, err := wav.DecodeWithoutResampling(file)
	if err != nil {
		fmt.Println("Error interpreting sound file: ", err)
	}
	soundPlay, err := context.NewPlayer(soundWav)
	if err != nil {
		fmt.Println("Couldn't create sound player: ", err)
	}
	return sound{soundPlay}
}

func (game *rpgGame) Update() error {
	getPlayerInput(game)

	game.player.animatePlayerSprite()
	game.animateDroppedItems()

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		game.player.action = INTERACT
	} else {
		game.player.action = WALK
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) && game.player.action == WALK {
		game.movePlayer(&game.player.xLoc)
	} else if ebiten.IsKeyPressed(ebiten.KeyD) && game.player.action == WALK {
		game.movePlayer(&game.player.xLoc)
	} else if ebiten.IsKeyPressed(ebiten.KeyW) && game.player.action == WALK {
		game.movePlayer(&game.player.yLoc)
	} else if ebiten.IsKeyPressed(ebiten.KeyS) && game.player.action == WALK {
		game.movePlayer(&game.player.yLoc)
	}
	game.outOfBoundsCheck()
	//fmt.Printf("x: %d, y: %d\n", game.player.xLoc, game.player.yLoc)
	game.itemsPickupCheck()
	if game.player.convertHeartItemsToHealth() {
		game.sounds.heal.playSound()
	}

	if game.player.action == INTERACT && game.player.interactCooldown < 0 {
		game.sounds.playerInteract.playSound()
		game.player.interactCooldown = COOLDOWN
		for i := range game.enemies {
			if game.enemies[i].level == game.levelCurrent {
				if game.player.playerInteractWithCharacterCheck(&game.enemies[i]) {
					game.enemies[i].hitPoints -= game.player.attackPower
					game.sounds.enemyHit.playSound()
					if game.enemies[i].hitPoints == 0 {
						game.enemies[i].death(game)
					}
				}
			}
		}
		if game.player.playerInteractWithCharacterCheck(&game.questGiver) && game.questGiver.level == game.levelCurrent {
			if game.player.questProgress == NOTTALKED {
				game.player.questProgress = TALKED
				//display quest text
				game.sounds.questGiverTalk.playSound()
			} else if game.player.questProgress == TALKED && game.player.questCheckInventoryForBook() { //AND H IASTEM
				game.player.questProgress = RETURNEDITEM
				game.player.attackPower++
				game.sounds.attackPowerUp.playSound()
			} else if game.player.questProgress == RETURNEDITEM {
				//display another thank you message?
			}
		}
	} else if game.player.interactCooldown > -10 {
		game.player.interactCooldown--
	}
	for i := range game.enemies {
		if game.enemies[i].action == PATH && game.enemies[i].pathUpdateCooldown < 0 &&
			game.enemies[i].level == game.levelCurrent {

			game.enemies[i].pathUpdateCooldown = COOLDOWN
			game.updatePath(&game.enemies[i], &game.player)

		} else if game.enemies[i].pathUpdateCooldown > -10 {
			game.enemies[i].pathUpdateCooldown--
		}
	}

	game.enemiesAttack()
	game.enemiesPathing()

	for i := range game.enemies {
		game.enemies[i].animateCharacter()
	}
	game.questGiver.animateCharacter()

	return nil
}

func (game *rpgGame) movePlayer(location *int) {
	if isBorderColliding(game.barrierRect, &game.player) {
		*location -= game.player.translateDirectionToPositiveNegative() * game.player.speed * 5
	} else {
		*location += game.player.translateDirectionToPositiveNegative() * game.player.speed
	}
}

func (game *rpgGame) outOfBoundsCheck() {
	if game.player.xLoc < -100 || game.player.xLoc > game.levelCurrent.TileWidth*game.levelCurrent.Width*worldScale {
		game.player.xLoc = 300
		game.player.yLoc = 300
	} else if game.player.yLoc < -100 || game.player.yLoc > game.levelCurrent.TileHeight*game.levelCurrent.Height*worldScale {
		game.player.xLoc = 300
		game.player.yLoc = 300
	}
}

func (game *rpgGame) Draw(screen *ebiten.Image) {
	//screen.Fill(colornames.Blue)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Reset()

	var teleporterIDs = []uint32{1, 2, 3}
	for _, layer := range game.levelCurrent.Layers {
		for tileY := 0; tileY < game.levelCurrent.Height; tileY++ {
			for tileX := 0; tileX < game.levelCurrent.Width; tileX++ {
				op.GeoM.Reset()
				//get on screen position
				tileXPos := float64(game.levelCurrent.TileWidth * tileX)
				tileYPos := float64(game.levelCurrent.TileHeight * tileY)
				op.GeoM.Translate(tileXPos, tileYPos)

				// Get the tile ID from the appropriate LAYER
				tileToDraw := layer.Tiles[tileY*game.levelCurrent.Width+tileX]

				if slices.Contains(game.barrierIDs, tileToDraw.ID) {
					barrierRectangle := image.Rect(int(tileXPos), int(tileYPos),
						int(tileXPos)+game.levelCurrent.TileWidth, int(tileYPos)+game.levelCurrent.TileHeight)
					game.barrierRect = append(game.barrierRect, barrierRectangle)
				}
				if slices.Contains(teleporterIDs, tileToDraw.ID) {
					teleporterRectangle := image.Rect(int(tileXPos), int(tileYPos),
						int(tileXPos)+game.levelCurrent.TileWidth, int(tileYPos)+game.levelCurrent.TileHeight)
					game.teleporterRects[tileToDraw.ID] = teleporterRectangle
				}

				if tileToDraw.ID != 0 {
					// Retrieve the corresponding sub-image from the map
					ebitenTileToDraw, ok := game.tileHashCurrent[tileToDraw.ID]
					if !ok {
						// Handle the case where the tile ID is not found in the map
						fmt.Printf("Tile ID %d not found in tileHashCurrent\n", tileToDraw.ID)
						continue
					}
					op.GeoM.Scale(worldScale, worldScale)
					// Draw the sub-image
					screen.DrawImage(ebitenTileToDraw, op)
				}
			}
		}
	}

	teleID := getTeleporterCollisionID(game.teleporterRects, &game.player)
	if teleID != 0 {
		game.changeWorldMap(teleID)
	}

	drawPlayerFromSpriteSheet(op, screen, game.player)
	for _, charact := range game.enemies {
		if charact.level == game.levelCurrent {
			drawImageFromSpriteSheet(op, screen, charact)
		}
	}
	if game.questGiver.level == game.levelCurrent {
		drawImageFromSpriteSheet(op, screen, game.questGiver)
	}

	for _, item := range game.droppedItems {
		if item.level == game.levelCurrent {
			op.GeoM.Reset()
			op.GeoM.Scale(resizeScale-1, resizeScale-1)
			op.GeoM.Translate(float64(item.xLoc), float64(item.yLoc-item.yAnimationOffset))
			screen.DrawImage(item.picture.(*ebiten.Image), op)
		}
	}

	game.drawPlayerHealth(op, screen)
	if game.questGiver.level == game.levelCurrent {
		switch game.player.questProgress {
		case TALKED:
			DrawCenteredText(screen, game.fontSmall, "My brother stole my book,\n   please get it back!",
				game.questGiver.xLoc+45, game.questGiver.yLoc)

		case RETURNEDITEM:
			DrawCenteredText(screen, game.fontSmall, "  Thank You!\nI've blessed you\n  with strength",
				game.questGiver.xLoc+45, game.questGiver.yLoc)
		}
	}

	DrawCenteredText(screen, game.fontSmall, "Power:", 50, 700)
	DrawCenteredText(screen, game.fontSmall, strconv.Itoa(game.player.attackPower), 120, 700)

	if game.player.hitPoints <= 0 {
		DrawCenteredText(screen, game.fontLarge, "GAME OVER", game.windowHeight/2, game.windowWidth/2)
	}
}

func drawPlayerFromSpriteSheet(op *ebiten.DrawImageOptions, screen *ebiten.Image, targetCharacter player) {
	op.GeoM.Reset()
	op.GeoM.Scale(resizeScale, resizeScale)
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
	if targetCharacter.direction == CHARACTLEFT {
		op.GeoM.Scale(resizeScale, resizeScale)
		op.GeoM.Translate(float64(targetCharacter.xLoc), float64(targetCharacter.yLoc))
	} else if targetCharacter.direction == CHARACTRIGHT {
		op.GeoM.Scale(-resizeScale, resizeScale)
		op.GeoM.Translate(
			float64(targetCharacter.xLoc)+(float64(targetCharacter.FRAME_WIDTH)*resizeScale), float64(targetCharacter.yLoc))
	}
	screen.DrawImage(targetCharacter.spriteSheet.SubImage(
		image.Rect(
			targetCharacter.frame*targetCharacter.FRAME_WIDTH,
			targetCharacter.imageYOffset*targetCharacter.FRAME_HEIGHT,
			targetCharacter.frame*targetCharacter.FRAME_WIDTH+targetCharacter.FRAME_WIDTH,
			targetCharacter.FRAME_HEIGHT+targetCharacter.FRAME_HEIGHT*targetCharacter.imageYOffset)).(*ebiten.Image), op)
}

func (game *rpgGame) drawPlayerHealth(op *ebiten.DrawImageOptions, screen *ebiten.Image) {
	op.GeoM.Reset()
	op.GeoM.Scale(worldScale, worldScale)
	for i := 0; i < game.player.hitPoints; i++ {
		screen.DrawImage(game.heartImage.(*ebiten.Image), op)
		op.GeoM.Translate(16*worldScale, 0)
	}
}

func (game *rpgGame) animateDroppedItems() {
	for i := range game.droppedItems {
		game.droppedItems[i].itemAnimate()
	}
}

func (game *rpgGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight //by default, just return the current dimensions
}

func main() {
	ebiten.SetWindowTitle("SimpleRPG")

	soundContext := audio.NewContext(soundSampleRate)

	sounds := sounds{
		enemyDeath:     loadEmbeddedWavToSound("enemyDeath.wav", soundContext),
		enemyHit:       loadEmbeddedWavToSound("enemyHit.wav", soundContext),
		attackPowerUp:  loadEmbeddedWavToSound("attackPowerUp.wav", soundContext),
		heal:           loadEmbeddedWavToSound("heal.wav", soundContext),
		playerInteract: loadEmbeddedWavToSound("playerInteract.wav", soundContext),
		playerDamaged:  loadEmbeddedWavToSound("playerDamaged.wav", soundContext),
		questGiverTalk: loadEmbeddedWavToSound("questGiverTalk.wav", soundContext),
		itemPickup:     loadEmbeddedWavToSound("itemPickup.wav", soundContext),
	}

	//tileMapHashes := make([]map[uint32]*ebiten.Image, 0, 5)
	//levelmaps := make([]*tiled.Map, 0, 5)
	//
	//gameMap := loadMapFromEmbedded(path.Join("assets", "dirt.tmx")) //0
	//ebitenImageMap := makeEbitenImagesFromMap(*gameMap)
	//levelmaps = append(levelmaps, gameMap)
	//tileMapHashes = append(tileMapHashes, ebitenImageMap)
	//
	//gameMap = loadMapFromEmbedded(path.Join("assets", "island.tmx")) //1
	//ebitenImageMap = makeEbitenImagesFromMap(*gameMap)
	//levelmaps = append(levelmaps, gameMap)
	//tileMapHashes = append(tileMapHashes, ebitenImageMap)
	//
	//gameMap = loadMapFromEmbedded(path.Join("assets", "world.tmx")) //2
	//ebitenImageMap = makeEbitenImagesFromMap(*gameMap)
	//levelmaps = append(levelmaps, gameMap)
	//tileMapHashes = append(tileMapHashes, ebitenImageMap)

	world := initializeWorldInfo()

	windowX := world.levelCurrent.TileWidth * world.levelCurrent.Width * worldScale
	windowY := world.levelCurrent.TileHeight * world.levelCurrent.Height * worldScale

	//windowX := gameMap.TileWidth * gameMap.Width * worldScale
	//windowY := gameMap.TileHeight * gameMap.Height * worldScale
	ebiten.SetWindowSize(windowX, windowY)
	fmt.Printf("windowWidth: %d, windowHeight: %d\n", windowX, windowY)

	playerSpriteSheet := LoadEmbeddedImage("characters", "player.png")
	enemySpriteSheet := LoadEmbeddedImage("characters", "characters.png")

	user := player{
		character: character{
			spriteSheet:      playerSpriteSheet,
			xLoc:             400,
			yLoc:             400,
			direction:        RIGHT,
			frame:            0,
			frameDelay:       0,
			FRAME_HEIGHT:     32,
			FRAME_WIDTH:      16,
			imageYOffset:     -1,
			speed:            3,
			hitPoints:        3,
			interactCooldown: COOLDOWN / 2,
			attackPower:      1,
		},
		questProgress: NOTTALKED,
	}
	questGiver := character{
		spriteSheet:      enemySpriteSheet,
		xLoc:             200,
		yLoc:             150,
		inventory:        nil,
		direction:        CHARACTLEFT,
		frame:            0,
		frameDelay:       0,
		FRAME_HEIGHT:     32,
		FRAME_WIDTH:      32,
		action:           WALK,
		imageYOffset:     0,
		level:            world.levelMaps[2],
		hitPoints:        1,
		interactCooldown: COOLDOWN,
	}

	mannequinInventory := make([]item, 0)
	mannequinInventory = append(mannequinInventory, BookItem)

	mannequin := character{
		spriteSheet:        enemySpriteSheet,
		xLoc:               100,
		yLoc:               100,
		inventory:          mannequinInventory,
		direction:          CHARACTLEFT,
		frame:              0,
		frameDelay:         0,
		FRAME_HEIGHT:       32,
		FRAME_WIDTH:        32,
		action:             PATH,
		imageYOffset:       0,
		speed:              2,
		level:              world.levelMaps[1],
		hitPoints:          2,
		interactCooldown:   COOLDOWN,
		attackPower:        1,
		pathUpdateCooldown: COOLDOWN,
	}

	king := character{
		spriteSheet:        enemySpriteSheet,
		xLoc:               100,
		yLoc:               200,
		inventory:          nil,
		direction:          CHARACTLEFT,
		frame:              0,
		frameDelay:         0,
		FRAME_HEIGHT:       32,
		FRAME_WIDTH:        32,
		action:             PATH,
		imageYOffset:       1,
		speed:              2,
		level:              world.levelMaps[0],
		hitPoints:          2,
		interactCooldown:   COOLDOWN,
		attackPower:        1,
		pathUpdateCooldown: COOLDOWN,
	}

	leprechaun := character{
		spriteSheet:        enemySpriteSheet,
		xLoc:               300,
		yLoc:               300,
		inventory:          nil,
		direction:          CHARACTRIGHT,
		frame:              0,
		frameDelay:         0,
		FRAME_HEIGHT:       32,
		FRAME_WIDTH:        32,
		action:             PATH,
		imageYOffset:       2,
		speed:              2,
		level:              world.levelMaps[0],
		hitPoints:          2,
		interactCooldown:   COOLDOWN,
		attackPower:        1,
		pathUpdateCooldown: COOLDOWN,
	}
	enemies := make([]character, 0, 5)
	enemies = append(enemies, mannequin)
	enemies = append(enemies, king)
	enemies = append(enemies, leprechaun)

	heartImage := grabItemImage(63, 0, 16, 16)
	droppedItems := make([]item, 0, 10)
	heart := HeartItem
	heart.level = world.levelMaps[2]
	droppedItems = append(droppedItems, heart)
	stone := StoneItem
	stone.level = world.levelMaps[2]
	droppedItems = append(droppedItems, stone)
	fmt.Printf("items: %d\n", droppedItems)

	teleporterRectangles := map[uint32]image.Rectangle{}

	var barrierID = []uint32{40, 41, 42, 43, 80, 81, 82, 83}

	game := rpgGame{
		//levelCurrent:    gameMap,
		//tileHashCurrent: ebitenImageMap,
		//levelMaps:       levelmaps,
		//tileHashes:      tileMapHashes,
		worldinfo:       *world,
		player:          user,
		enemies:         enemies,
		barrierIDs:      barrierID,
		windowWidth:     windowX,
		windowHeight:    windowY,
		teleporterRects: teleporterRectangles,
		heartImage:      heartImage,
		fontLarge:       LoadScoreFont(60),
		fontSmall:       LoadScoreFont(16),
		droppedItems:    droppedItems,
		questGiver:      questGiver,
		sounds:          sounds,
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
	for _, layer := range tiledMap.Layers {
		for _, tile := range layer.Tiles {

			if _, ok := idToImage[tile.ID]; !ok { //if tileID does not exists
				x := int((tile.ID)%uint32(tiledMap.Tilesets[0].Columns)) * tiledMap.TileWidth
				y := int((tile.ID)/uint32(tiledMap.Tilesets[0].Columns)) * tiledMap.TileHeight
				subImageRect := image.Rect(x, y, x+tiledMap.TileWidth, y+tiledMap.TileHeight)
				subImage := ebitenImageTileset.SubImage(subImageRect).(*ebiten.Image)
				idToImage[tile.ID] = subImage
			} else {
				//do nothing?
			}
		}
	}

	return idToImage
}

func grabItemImage(startX, startY, width, height int) image.Image {
	spriteSheet := LoadEmbeddedImage("", "objects.png")
	subImageRect := image.Rect(startX, startY, startX+width, startY+height)

	subImage := ebiten.NewImageFromImage(spriteSheet).SubImage(subImageRect)
	return subImage
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

func LoadScoreFont(size float64) font.Face {
	//originally inspired by https://www.fatoldyeti.com/posts/roguelike16/
	trueTypeFont, err := opentype.Parse(fonts.PressStart2P_ttf)
	if err != nil {
		fmt.Println("Error loading font for score:", err)
	}
	fontFace, err := opentype.NewFace(trueTypeFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		fmt.Println("Error loading font of correct size for score:", err)
	}
	return fontFace
}

func DrawCenteredText(screen *ebiten.Image, font font.Face, s string, cx, cy int) { //from https://github.com/sedyh/ebitengine-cheatsheet
	bounds := text.BoundString(font, s)
	x, y := cx-bounds.Min.X-bounds.Dx()/2, cy-bounds.Min.Y-bounds.Dy()/2
	text.Draw(screen, s, font, x, y, colornames.White)
}

func isBorderColliding(borderRects []image.Rectangle, player *player) bool {
	playerBounds := player.getCollisionBoundingBox()

	for _, border := range borderRects {
		borderBounds := collision.BoundingBox{
			X:      float64(border.Min.X * worldScale),
			Y:      float64(border.Min.Y * worldScale),
			Width:  float64(border.Dx() * worldScale),
			Height: float64(border.Dy() * worldScale),
		}
		if collision.AABBCollision(playerBounds, borderBounds) {
			return true
		}
	}
	return false
}

func getTeleporterCollisionID(teleporterRects map[uint32]image.Rectangle, player *player) uint32 {
	playerBounds := player.getCollisionBoundingBox()

	for ID, teleporter := range teleporterRects {
		teleporterBounds := collision.BoundingBox{
			X:      float64(teleporter.Min.X * worldScale),
			Y:      float64(teleporter.Min.Y * worldScale),
			Width:  float64(teleporter.Dx() * worldScale),
			Height: float64(teleporter.Dy() * worldScale),
		}
		if collision.AABBCollision(playerBounds, teleporterBounds) {
			return ID
		}
	}
	return 0
}

func (game *rpgGame) changeWorldMap(tileID uint32) {
	//
	if tileID == 1 {
		// go to right world
		game.levelCurrent = game.levelMaps[1]
		game.tileHashCurrent = game.tileHashes[1]
		game.player.xLoc = 50
		game.pathGridCurrent = game.pathGrids[1]
	} else if tileID == 2 {
		// go to main world
		game.levelCurrent = game.levelMaps[2]
		game.tileHashCurrent = game.tileHashes[2]
		if game.player.xLoc > 600 {
			game.player.xLoc = 50
		} else if game.player.xLoc < 150 {
			game.player.xLoc = game.windowWidth - 100
		}
		game.pathGridCurrent = game.pathGrids[2]
	} else if tileID == 3 {
		//go to left world
		game.levelCurrent = game.levelMaps[0]
		game.tileHashCurrent = game.tileHashes[0]
		game.player.xLoc = game.windowWidth - 100
		game.pathGridCurrent = game.pathGrids[0]
	}
	//fmt.Println(game.levelCurrent)
	game.barrierRect = game.barrierRect[:0]
	game.teleporterRects = make(map[uint32]image.Rectangle)
}

func (game *rpgGame) enemiesAttack() {
	for i := range game.enemies {
		if game.enemies[i].isPlayerInAttackRange(&game.player) && game.enemies[i].interactCooldown < 0 {
			if game.enemies[i].level == game.levelCurrent {
				//damage player
				game.sounds.playerDamaged.playSound()
				game.player.hitPoints -= game.enemies[i].attackPower
				game.enemies[i].interactCooldown = COOLDOWN
			}
		} else if game.enemies[i].interactCooldown > -10 {
			game.enemies[i].interactCooldown--
		}
	}
}

func (game *rpgGame) enemiesPathing() {
	for i := range game.enemies {
		if game.enemies[i].level == game.levelCurrent {
			game.moveCharacterAlongPath(&game.enemies[i])
		}
	}
}

func (game *rpgGame) itemsPickupCheck() {
	newDroppedItems := make([]item, 0)
	for i := range game.droppedItems {
		if game.player.isItemColliding(&game.droppedItems[i]) && game.droppedItems[i].level == game.levelCurrent {
			game.player.inventory = append(game.player.inventory, game.droppedItems[i])
			if game.droppedItems[i].displayName != "Heart" {
				game.sounds.itemPickup.playSound()
			}
		} else {
			newDroppedItems = append(newDroppedItems, game.droppedItems[i])
		}
	}
	game.droppedItems = newDroppedItems
}

func (game *rpgGame) updatePath(c *character, player *player) {
	cStartCol := c.xLoc / game.levelCurrent.TileWidth
	cStartRow := c.yLoc / game.levelCurrent.TileHeight
	playerCol := player.xLoc / game.levelCurrent.TileWidth
	playerRow := player.yLoc / game.levelCurrent.TileHeight

	startCell := game.pathGridCurrent.Get(cStartCol, cStartRow)
	endCell := game.pathGridCurrent.Get(playerCol, playerRow)

	c.path = game.pathGridCurrent.GetPathFromCells(startCell, endCell, false, false)

}

func (game *rpgGame) moveCharacterAlongPath(c *character) {
	if c.path != nil {
		pathCell := c.path.Current()
		if math.Abs(float64(pathCell.X*game.levelCurrent.TileWidth)-float64(c.xLoc)) <= 2 &&
			math.Abs(float64(pathCell.Y*game.levelCurrent.TileHeight)-float64(c.yLoc)) <= 2 { //if we are now on the tile we need to be on
			c.path.Advance()
		}
		direction := 0
		if pathCell.X*game.levelCurrent.TileWidth > c.xLoc {
			direction = 1
		} else if pathCell.X*game.levelCurrent.TileWidth < c.xLoc {
			direction = -1
		}
		Ydirection := 0
		if pathCell.Y*game.levelCurrent.TileHeight > c.yLoc {
			Ydirection = 1
		} else if pathCell.Y*game.levelCurrent.TileHeight < c.yLoc {
			Ydirection = -1
		}
		c.xLoc += direction * c.speed
		c.yLoc += Ydirection * c.speed
	}
}
