CREATE TABLE player (
    ID VARCHAR(255) PRIMARY KEY NOT NULL,
    Name VARCHAR(255) NOT NULL,
    PictureURL VARCHAR(255) NOT NULL
);

CREATE TABLE rank (
    GameName VARCHAR(255) NOT NULL,
    GameDate TIMESTAMP NOT NULL,
	PlayerID VARCHAR(255) NOT NULL REFERENCES player(ID),
	Points INTEGER NOT NULL,

    UNIQUE (GameName, GameDate, PlayerID)
);

DROP TABLE rank;

SELECT * FROM pg_catalog.pg_tables;
SELECT * FROM "player" LIMIT 1000;
SELECT * FROM "rank" LIMIT 1000;

SELECT playerid, sum(points) as total
FROM rank
GROUP BY playerid
ORDER BY total DESC;

SELECT gamename, sum(points) as total
FROM rank
GROUP BY gamename
ORDER BY total DESC;

SELECT gamename, playerid, sum(points) as total
FROM rank
GROUP BY gamename, playerid
ORDER BY total DESC;