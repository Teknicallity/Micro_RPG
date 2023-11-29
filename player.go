package main

import (
	"fmt"
	"github.com/co0p/tankism/lib/collision"
	"image"
)

type player struct {
	character
	questProgress int
}

func (player *player) playerInteractWithCharacterCheck(target *character) bool {
	player.updatePlayerInteractionRectangle()
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

func (player *player) updatePlayerInteractionRectangle() {
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

func (player *player) getInventoryItemIndex(itemName string) int {
	for i := range player.inventory {
		if player.inventory[i].displayName == itemName {
			return i
		}
	}
	return -1
}

// player
func (player *player) questCheckInventoryForBook() bool {
	index := player.getInventoryItemIndex(BookItem.displayName)
	if index != -1 {
		player.removeInventoryItemAtIndex(index)
		return true
	}
	return false
}

func (player *player) convertHeartItemsToHealth() bool {
	indexToRemove := 0
	remove := false
	for i := range player.inventory {
		if player.inventory[i].displayName == "Heart" {
			indexToRemove = i
			remove = true
			player.hitPoints++
			break
		}
	}
	if remove {
		player.removeInventoryItemAtIndex(indexToRemove)
	}
	return remove
}

func (player *player) animatePlayerSprite() {
	if player.action == WALK {
		player.frameDelay += 1
		if player.frameDelay%8 == 0 {
			player.frame += 1
			if player.frame >= 4 {
				player.frame = 0
			}
		}
	} else if player.action == INTERACT {
		if 4 <= player.frame && player.frame <= 7 {
			player.frameDelay += 1
			if player.frameDelay%8 == 0 {
				player.frame--
				if player.frame <= 4 {
					player.frame = 7
				}
			}
		} else {
			player.frame = 7
		}
	}
}

func (player *player) translateDirectionToPositiveNegative() int {
	switch player.direction {
	case DOWN:
		return 1
	case RIGHT:
		return 1
	case UP:
		return -1
	case LEFT:
		return -1
	default:
		return 0
	}
}
