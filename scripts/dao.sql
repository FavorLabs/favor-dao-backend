use dao

db.createCollection("dao");
db.createCollection("dao_bookmark");
db.createCollection("d_user");
db.createCollection("tag");
db.createCollection("post");
db.createCollection("post_content");
db.createCollection("post_star");
db.createCollection("post_collection");
db.createCollection("comment");
db.createCollection("comment_content");
db.createCollection("comment_reply");

db.dao.createIndexes([
    {address: 1},
]);

db.dao.createIndex({"name": "text"}, {unique: true})

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

db.tag.createIndex({tag: 1}, {unique: true});
db.tag.createIndexes([
    {address: 1}
]);

db.user.createIndex({"address": 1}, {unique: true})

db.user.createIndex({nickname: "text"}, {unique: true});

db.comment.createIndexes([
    {post_id: 1},
    {address: 1}
]);

db.comment_content.createIndexes([
    {comment_id: 1},
    {address: 1}
]);

db.comment_reply.createIndexes([
    {comment_id: 1},
    {address: 1}
]);