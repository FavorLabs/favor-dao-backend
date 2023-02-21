use dao

db.createCollection("dao");
db.createCollection("dao_bookmark");
db.createCollection("user");
db.createCollection("tag");
db.createCollection("post");
db.createCollection("post_content");
db.createCollection("post_star");
db.createCollection("post_collection");


db.dao.createIndexes([
    {address: 1},
    {name: "text"},
]);

db.dao_bookmark.createIndexes([
    {dao_id: 1},
    {address: 1}
]);

db.post_collection.createIndexes([
    {post_id: 1},
    {address: 1}
]);

db.post_content.createIndexes([
    {post_id: 1},
    {address: 1}
]);

db.post_star.createIndexes([
    {post_id: 1},
    {address: 1}
]);

db.post.createIndexes([
    {dao_id: 1},
    {address: 1}
]);

db.tag.createIndexes([
    {tag: "text"},
    {address: 1}
]);

db.user.createIndex({"address": 1}, {unique: true})

db.user.createIndex({nickname: "text"});