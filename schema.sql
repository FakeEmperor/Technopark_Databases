CREATE TABLE User (
  id                          INT AUTO_INCREMENT PRIMARY KEY NOT NULL,
  about                       TEXT UNICODE,
  email                       VARCHAR(255) NOT NULL UNIQUE,
  status_is_anonymous         BOOL NOT NULL,
  name                        VARCHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  username                    VARCHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci

) ENGINE = InnoDB;

CREATE INDEX email_idx ON User (email);

CREATE TABLE Forum(
  id                          INT AUTO_INCREMENT PRIMARY KEY NOT NULL,
  name                        VARCHAR(255) UNICODE UNIQUE,
  short_name                  VARCHAR(64) UNIQUE,
  user                        VARCHAR(255),
  FOREIGN KEY (user)          REFERENCES User(email) ON DELETE SET NULL
) ENGINE = InnoDB;

CREATE INDEX short_name_idx ON Forum (short_name);


CREATE TABLE Message(
  id                          INT AUTO_INCREMENT PRIMARY KEY NOT NULL,
  date                        DATETIME NOT NULL,
  status_is_deleted           BOOL NOT NULL DEFAULT 0,
  message                     TEXT UNICODE NOT NULL,
  forum                       VARCHAR(64) NOT NULL,
  user                        VARCHAR(255), -- can be null, but default is not null
  FOREIGN KEY (forum)         REFERENCES Forum(short_name) ON DELETE CASCADE,
  FOREIGN KEY (user)          REFERENCES User(email) ON DELETE SET NULL,
  INDEX (date)
) ENGINE = InnoDB;


CREATE TABLE Thread(
  id                          INT PRIMARY KEY NOT NULL,
  status_is_closed            BOOL DEFAULT 0,
  slug                        VARCHAR(128) UNIQUE, -- can be null after delete
  title                       VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  FOREIGN KEY (id)            REFERENCES Message(id) ON DELETE CASCADE
) ENGINE = InnoDB;

CREATE TABLE Post(
  id                          INT PRIMARY KEY NOT NULL,
  status_is_approved          BOOL NOT NULL DEFAULT 1, -- need to check on docs ( is default null / false / true )?
  status_is_edited            BOOL NOT NULL DEFAULT 0, -- need to check on docs ( is default null / false / true )?
  status_is_highlighted       BOOL NOT NULL DEFAULT 0, -- need to check on docs ( is default null / false / true )?
  status_is_spam              BOOL NOT NULL DEFAULT 0, -- need to check on docs ( is default null / false / true )?
  parent_id                   INT, -- default is null
  thread_id                   INT NOT NULL,
  material_path               VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  FOREIGN KEY (thread_id)     REFERENCES Thread(id) ON DELETE CASCADE,
  FOREIGN KEY (id)            REFERENCES Message(id) ON DELETE CASCADE
) ENGINE = InnoDB;
-- M2M connection for posts (user rate<-post)
CREATE TABLE UserMessageRate(
  like_id                     INT PRIMARY KEY AUTO_INCREMENT,
  message_id                  INT NOT NULL, -- this is stupid, why no user?
  status_is_rate_like         BOOL NOT NULL, -- like/dislike
  FOREIGN KEY (message_id)       REFERENCES Message(id) ON DELETE CASCADE
) ENGINE = InnoDB;
-- M2M connection for users and threads (user->thread)
CREATE TABLE UserSubscription (
  user                        VARCHAR(255) NOT NULL,
  thread_id                   INT NOT NULL,
  FOREIGN KEY (user)          REFERENCES User(email) ON DELETE CASCADE,
  FOREIGN KEY (thread_id)     REFERENCES Thread(id) ON DELETE CASCADE,
  PRIMARY KEY (user, thread_id),
  INDEX (user)
) ENGINE = InnoDB;
-- M2M connection for users (follower->followee)
CREATE TABLE UserFollowers(
  follower VARCHAR(255) NOT NULL,
  followee VARCHAR(255) NOT NULL,
  FOREIGN KEY (follower)   REFERENCES User(email) ON DELETE CASCADE ,
  FOREIGN KEY (followee)   REFERENCES User(email) ON DELETE CASCADE ,
  PRIMARY KEY (followee, follower),
  INDEX (followee)
) ENGINE = InnoDB;