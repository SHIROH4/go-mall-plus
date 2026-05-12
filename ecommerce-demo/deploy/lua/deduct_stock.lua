-- KEYS[1]: 扣减库存的 Redis key (例如: stock:1)
-- ARGV[1]: 想要扣减的数量 (比如买 1 件)

local stockKey = KEYS[1]
local deductCount = tonumber(ARGV[1])

-- 1. 获取当前库存
local currentStock = redis.call('GET', stockKey)

-- 2. 如果库存为空，说明没初始化，返回 -1 表示异常
if currentStock == false then
    return -1
end

currentStock = tonumber(currentStock)

-- 3. 判断库存够不够
if currentStock >= deductCount then
    -- 4. 够，则扣减并返回 1 表示扣减成功
    redis.call('DECRBY', stockKey, deductCount)
    return 1
else
    -- 5. 不够，返回 0 表示库存不足
    return 0
end