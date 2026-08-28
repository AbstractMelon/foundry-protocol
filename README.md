# Foundry Protocol

A persistent multiplayer factory-building and combat RTS (Factorio + Mindustry vibes).
Players run a 24/7 world server, then connect with a client (browser or desktop)
to build factories, defend them, and fight to be the last base standing.

## Prerequisites

- Go 1.26+
- Godot 4.5+
- `air` for hot reload

## Quickstart

```bash
./scripts/dev.sh --client
```

This boots a world server in **dev mode** and the gateway with hot reload, then opens
the Godot client.

Dev-mode chat commands (server-side, any player):

```
/give copper 500     # add resources
/set iron 10         # set a resource to a fixed amount
/clear               # remove every building in the world
/help
```

## Making content

Nothing gameplay-y is hardcoded. The world server loads **all** content from
`content/*.yaml` at boot and sends its definitions to the client over the wire, so the
client UI renders whatever exists.

### Textures / art

Drop a PNG (or JPG/WebP) into `assets/` and reference it from any content entry with
its filename:

```yaml
texture: iron.png
```

The world server embeds referenced textures (as base64 data URLs) into the content bundle it sends every client, so the client always draws whatever art you add. Textures are matched case-insensitively (`Iron.png` and `iron.png` are the same file).

### Resources & ore deposits

A resource deposit is a transparent texture layered **on top** of a normal terrain tile, and mining it away leaves the base block behind. A resource becomes a mineable deposit when it has both `can_place_on` (which
base terrains it may spawn on) and a `yield` (how much it holds):

```yaml
resources:
  - id: iron
    name: Iron
    color: "#7f8fa6"
    stack_size: 50
    texture: iron.png
    can_place_on: [grass, rock, sand]
    yield: 600
```

### Recipes & buildings

Add a recipe (production rule) to `content/recipes.yaml`:

```yaml
recipes:
  - id: smelt_titanium
    name: Smelt Titanium
    category: smelting
    input: { titanium: 2 }
    output: { titanium_bar: 1 }
    duration_ticks: 20
```

Add a building that uses it to `content/buildings.yaml`:

```yaml
buildings:
  - id: alloyer
    name: Alloyer
    category: production
    color: "#9c88ff"
    health: 300
    cost: { copper_bar: 30 }
    recipe: smelt_titanium
    storage: { titanium: 100, titanium_bar: 100 }
```

Save the file and the running dev world server rebuilds itself.
