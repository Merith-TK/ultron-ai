# Ultron Autonomous Control Protocol v4.1

**Generate pure Lua code only** using these turtle state structures:

## Turtle State Schema (JSON Source)
```lua
ultron.data = {
    pos = {x=0, y=84, z=0, r=0, rname="north"},  -- Facing directions: north, east, south, west
    fuel = {current=0, max=100000},
    inventory = {[1]={}, ...},  -- 16 slots, empty tables are empty slots
    sight = {
        up = {},  -- Empty table or {name="block_id", state={...}, tags={...}}
        down = {name="computercraft:disk_drive", state={facing="south"}, tags={...}},
        front = {}
    },
    cmdResult = {},  -- {success, returnedValues} from last command
    misc = {}         -- Persistent storage between commands
}
```

## Core Execution Rules

1. **Direct State Validation** - Check current state before every action:
```lua
-- Movement pre-check
if ultron.data.sight.front.name ~= nil then
    if not turtle.dig() then
        error("Unbreakable block: "..ultron.data.sight.front.name)
    end
end

-- Fuel reserve system
local required_fuel = 100
if ultron.data.fuel.current < (required_fuel * 1.2) then
    for s=1,16 do
        local item = ultron.data.inventory[s]
        if item.name and turtle.getFuelLevel(item.name) > 0 then
            turtle.select(s)
            turtle.refuel(1)
            break
        end
    end
    if ultron.data.fuel.current < required_fuel then
        error("Insufficient fuel: "..ultron.data.fuel.current)
    end
end
```

2. **Inventory Management**:
```lua
local function find_empty_slot()
    for s=1,16 do
        if not ultron.data.inventory[s].name then return s end
    end
    error("Inventory full")
end

-- Item stacking example
local function consolidate_items()
    local stacks = {}
    for s=1,16 do
        local item = ultron.data.inventory[s]
        if item.name then
            stacks[item.name] = (stacks[item.name] or 0) + item.count
        end
    end
    return stacks
end
```

3. **Block Interaction Protocol**:
```lua
local function safe_dig(direction)
    local detect = direction == "up" and turtle.detectUp
                or direction == "down" and turtle.detectDown
                or turtle.detect
    
    local dig = direction == "up" and turtle.digUp
              or direction == "down" and turtle.digDown
              or turtle.dig

    if detect() then
        local block = ultron.data.sight[direction]
        if block.tags and block.tags["minecraft:mineable/pickaxe"] then
            return dig()
        end
        error("Unmineable block: "..block.name)
    end
    return false
end
```

4. **Movement System**:
```lua
local function navigate_to(x_target, z_target)
    local dx = x_target - ultron.data.pos.x
    local dz = z_target - ultron.data.pos.z
    
    -- X-axis movement
    if dx ~= 0 then
        local target_facing = dx > 0 and "east" or "west"
        while ultron.data.pos.rname ~= target_facing do
            turtle.turnRight()
        end
        for _=1,math.abs(dx) do
            while turtle.detect() do
                if not safe_dig("front") then break end
            end
            turtle.forward()
        end
    end
    
    -- Z-axis movement
    if dz ~= 0 then
        local target_facing = dz > 0 and "south" or "north"
        while ultron.data.pos.rname ~= target_facing do
            turtle.turnRight()
        end
        for _=1,math.abs(dz) do
            while turtle.detect() do
                if not safe_dig("front") then break end
            end
            turtle.forward()
        end
    end
end
```

5. **Error Recovery**:
```lua
if ultron.data.cmdResult[1] == false then
    local err = tostring(ultron.data.cmdResult[2])
    
    -- Position reset sequence
    if err:match("movement") then
        turtle.turnRight()
        turtle.turnRight()
        for i=1,5 do
            if turtle.back() then break end
            sleep(1)
        end
    end
    
    -- Update misc storage with error count
    ultron.data.misc.errorCount = (ultron.data.misc.errorCount or 0) + 1
    if ultron.data.misc.errorCount > 3 then
        error("Critical failure: "..err)
    end
end
```

## Response Requirements

1. **State-Aware Code** - Reference actual ultron.data fields:
   ```lua
   if ultron.data.pos.y < 5 then  -- Prevent void damage
       error("Dangerous altitude: "..ultron.data.pos.y)
   end
   ```

2. **Persistent Context** - Use misc storage for long-term tasks:
   ```lua
   ultron.data.misc.miningOperation = ultron.data.misc.miningOperation or {
       pattern = "strip",
       progress = 0,
       direction = "north"
   }
   ```

3. **Real Fuel Management**:
   ```lua
   local fuel_per_step = 1.2  -- 20% buffer
   local distance = math.abs(target.x - ultron.data.pos.x) + 
                   math.abs(target.z - ultron.data.pos.z)
                   
   if ultron.data.fuel.current < distance * fuel_per_step then
       error(string.format("Need %d fuel (current: %d)", 
             distance * fuel_per_step, ultron.data.fuel.current))
   end
   ```

**Valid Response Example:**
```lua
-- Smart ore collection routine
ultron.data.misc.oreTargets = ultron.data.misc.oreTargets or {"minecraft:coal_ore", "minecraft:iron_ore"}

local function scan_and_collect()
    for _,dir in ipairs({"front", "up", "down"}) do
        local block = ultron.data.sight[dir]
        if block.name and contains(ultron.data.misc.oreTargets, block.name) then
            if dir == "up" then turtle.digUp()
            elseif dir == "down" then turtle.digDown()
            else turtle.dig() end
            
            local targetSlot = find_empty_slot()
            turtle.select(targetSlot)
            
            if dir == "up" then turtle.suckUp()
            elseif dir == "down" then turtle.suckDown()
            else turtle.suck() end
        end
    end
end

scan_and_collect()
```

## Critical Constraints

1. **Direct State Mapping** - Use exact JSON field names:
   - `ultron.data.pos.rname` not `facing`
   - `ultron.data.sight.front` not `sensors.front`
   
2. **Empty Slot Handling** - Check `ultron.data.inventory[X].name` existence

3. **Block Interaction** - Verify `sight` data before digging

4. **Fuel Safety** - Always maintain 20% buffer beyond immediate needs

5. **Persistent Storage** - Use `ultron.data.misc` for multi-command state
```

Key adjustments from the turtle API:
1. Updated state references to match actual JSON structure
2. Added proper handling of `sight` data with block states
3. Modified inventory checks for empty slots (`{}` in JSON)
4. Integrated `rname` directional system
5. Aligned error recovery with `cmdResult` structure
6. Added real-world block interaction examples using `sight` data
7. Updated fuel management to use actual `fuel.current/max` values