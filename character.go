package main

import (
	"fmt"
	"github.com/co0p/tankism/lib/collision"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/lafriks/go-tiled"
	"image"
)

type character struct {
	spriteSheet      *ebiten.Image
	xLoc             int
	yLoc             int
	hitPoints        int
	inventory        []item
	direction        int
	frame            int
	frameDelay       int
	FRAME_HEIGHT     int
	FRAME_WIDTH      int
	action           int
	imageYOffset     int
	speed            int
	level            *tiled.Map
	interactRect     image.Rectangle
	interactCooldown int
	questProgress    int
	attackPower      int
}

func (player *character) playerInteractWithCheck(target *character) bool {
	player.updatePlayerInteractRectangle()
	fmt.Printf("%d", player.direction)
	fmt.Printf("player X: %d, Y: %d\n", player.xLoc, player.yLoc)
	fmt.Printf("itneractbox X: %d, Y: %d  width: %d, height: %d\n", player.interactRect.Min.X, player.interactRect.Min.Y, player.interactRect.Dx()*worldScale, player.interactRect.Dy()*worldScale)
	targetBounds := target.getCollisionBoundingBox()
	playerBounds := player.getCollisionBoundingBox()
	playerInteractBounds := collision.BoundingBox{
		X:      float64(player.interactRect.Min.X),
		Y:      float64(player.interactRect.Min.Y),
		Width:  float64(player.interactRect.Dx()),
		Height: float64(player.interactRect.Dy()),
	}
	if collision.AABBCollision(playerBounds, targetBounds) || collision.AABBCollision(playerInteractBounds, targetBounds) {
		return true
	}
	return false
}

func (player *character) updatePlayerInteractRectangle() {
	//based on direction, change targetRectangle
	switch player.direction {
	case DOWN:
		player.interactRect = image.Rect(
			player.xLoc,
			player.yLoc+player.FRAME_HEIGHT*resizeScale,
			player.xLoc+player.FRAME_WIDTH*resizeScale,
			player.yLoc+player.FRAME_HEIGHT*resizeScale+player.FRAME_WIDTH,
		)
	case RIGHT:
		player.interactRect = image.Rect(
			player.xLoc+player.FRAME_WIDTH*resizeScale,
			player.yLoc,
			player.xLoc+(player.FRAME_WIDTH*resizeScale*2),
			player.yLoc+player.FRAME_HEIGHT,
		)
	case UP:
		player.interactRect = image.Rect(
			player.xLoc,
			player.yLoc,
			player.xLoc+(player.FRAME_WIDTH*resizeScale),
			player.yLoc-(player.FRAME_WIDTH*resizeScale)-player.FRAME_WIDTH,
		)
	case LEFT:
		player.interactRect = image.Rect(
			player.xLoc,
			player.yLoc,
			player.xLoc-player.FRAME_WIDTH*resizeScale,
			player.yLoc+player.FRAME_HEIGHT,
		)
	}
}

func (npc *character) isPlayerInAttackRange(player *character) bool {
	player.updatePlayerInteractRectangle()
	npcBounds := npc.getCollisionBoundingBox()
	playerBounds := player.getCollisionBoundingBox()

	if collision.AABBCollision(playerBounds, npcBounds) {
		return true
	}
	return false
}

func (character *character) death(game *rpgGame) {
	character.dropAllItems(game)
	character.xLoc = -100
	character.yLoc = -100
	game.sounds.enemyDeath.playSound()
}

func (character *character) isItemColliding(item *item) bool {
	itemBounds := collision.BoundingBox{
		X:      float64(item.xLoc),
		Y:      float64(item.yLoc),
		Width:  float64(item.picture.Bounds().Dx() * resizeScale),
		Height: float64(item.picture.Bounds().Dy() * resizeScale),
	}
	playerBounds := character.getCollisionBoundingBox()

	if collision.AABBCollision(itemBounds, playerBounds) {
		return true
	} else {
		return false
	}
}

func (character *character) getCollisionBoundingBox() collision.BoundingBox {
	boundBox := collision.BoundingBox{
		X:      float64(character.xLoc),
		Y:      float64(character.yLoc),
		Width:  float64(character.FRAME_WIDTH * resizeScale),
		Height: float64(character.FRAME_HEIGHT * resizeScale),
	}
	return boundBox
}

func (character *character) dropAllItems(game *rpgGame) {
	for i := range character.inventory {
		character.dropItem(game, i)
	}
	character.dropItem(game, -1)
}

func (character *character) dropItem(game *rpgGame, itemIndex int) {
	//character.inventory[itemIndex] = nil
	if itemIndex < 0 {
		heart := HeartItem
		heart.xLoc = character.xLoc + 20
		heart.yLoc = character.yLoc + 20
		heart.level = character.level
		game.droppedItems = append(game.droppedItems, heart)
	} else {
		droppedItem := character.inventory[itemIndex]
		droppedItem.xLoc = character.xLoc + 40
		droppedItem.yLoc = character.yLoc + 40
		droppedItem.level = character.level
		game.droppedItems = append(game.droppedItems, droppedItem)
		character.removeInventoryItemAtIndex(itemIndex)
	}
}

func (character *character) getItemIndex(itemName string) int {
	for i := range character.inventory {
		if character.inventory[i].displayName == itemName {
			return i
		}
	}
	return -1
}

func (character *character) questCheckForBook() bool {
	index := character.getItemIndex(BookItem.displayName)
	if index != -1 {
		character.removeInventoryItemAtIndex(index)
		return true
	}
	return false
}

func (character *character) removeInventoryItemAtIndex(index int) {
	retained := make([]item, 0)
	for i := range character.inventory {
		if i != index {
			retained = append(retained, character.inventory[i])
		}
	}
	character.inventory = retained
}

func (character *character) convertHeartItemsToHealth() bool {
	indexToRemove := 0
	remove := false
	for i := range character.inventory {
		if character.inventory[i].displayName == "Heart" {
			indexToRemove = i
			remove = true
			character.hitPoints++
			break
		}
	}
	if remove {
		character.removeInventoryItemAtIndex(indexToRemove)
	}
	return remove
}
