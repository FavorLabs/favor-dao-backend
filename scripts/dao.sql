use dao;

db.createCollection("dao");
db.createCollection("dao_bookmark");
db.createCollection("user");
// db.createCollection("user_setting");
db.createCollection("tag");
db.createCollection("post");
db.createCollection("post_content");
db.createCollection("post_star");
db.createCollection("post_collection");

db.dao.insertOne({
    "name": "",
    "address": "0x000",
    "visibility": 0, // 可见性 0公开 1私密
    "introduction": "简介",
    "is_del": 0
})

db.dao_bookmark.insertOne({
    "dao_id": "",
    "address": "0x000",
    "is_del": 0
})

db.post_star.insertOne({
    "post_id": "",
    "address": "0x000",
    "is_del": 0
})

db.post_collection.insertOne({
    "post_id": "",
    "address": "0x000",
    "is_del": 0
})

db.post_content.insertOne({
    "post_id": "",
    "address": "0x000",
    "content": "",
    "type": 2, // 类型，1标题，2文字段落，3图片地址，4视频地址，5语音地址，6链接地址，7封面图，8 favor资源
    "sort": 100, // 排序，越小越靠前
    "is_del": 0
})

db.post.insertOne({
    "dao_id": "",
    "type": 0, // 分类， 1 简讯 2 视频
    "address": "0x000",
    "visibility": 0, // 0 draft 1 'private', 2'public', 3'secret'//不聚合
    "member": 0, // level
    "is_top": 0,
    "is_essence": 0, // 是否精华
    "collection_count": 0, // 收藏数
    "upvote_count": 0, // 点赞数
    "view_count": 0, // 观看数
    // "is_lock": 0,  // 是否锁定
    "tags": "",
    "is_del": 0
})

db.user.insertOne({
    "address": "0x123456789",
    "nickname": "test-A",
    "is_del": 0,
    "avatar": "",
    "role": ""
})

db.tag.insertOne({
    "address": "0x000",
    "tag": "",
    "quote_num": 0,
    "is_del": 0,
})

db.user.updateMany({"address": "123"}, {
    $set: {"nickname": "paopao", "is_admin": 1, "is_del": 0},
    $currentDate: {lastModified: true}
})

db.user.find({_id: new ObjectId("63ec77bb1f074056c82d3273")})
