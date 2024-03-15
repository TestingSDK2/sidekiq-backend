CREATE TABLE `User` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `userName` varchar(80) DEFAULT NULL,
  `firstname` varchar(120) DEFAULT NULL,
  `lastname` varchar(120) DEFAULT NULL,
  `email` varchar(120) DEFAULT NULL,
  `password` varchar(60) DEFAULT NULL,
  `createDate` datetime NOT NULL DEFAULT '2018-01-01 00:00:00',
  `lastModifiedDate` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `userName` (`userName`)
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8;

CREATE TABLE `Contact` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `ownerID` int(10) unsigned NOT NULL,
  `userID` int(10) unsigned NULL,
  `firstName` varchar(30) DEFAULT NULL,
  `lastName` varchar(30) DEFAULT NULL,
  `address` varchar(50) DEFAULT NULL,
  `city` varchar(50) DEFAULT NULL,
  `state` varchar(50) DEFAULT NULL,
  `zip` varchar(10) DEFAULT NULL,
  `country` varchar(30) NOT NULL DEFAULT '',
  `phone` varchar(15) DEFAULT NULL,
  `fax` varchar(15) DEFAULT NULL,
  `email` varchar(100) DEFAULT NULL,
  `lastModifiedDate` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `Contact_ownerID` (`ownerID`),
  KEY `Contact_userID` (`userID`),
  KEY `Contact_email` (`email`)
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8;

CREATE TABLE `Organization` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `company` varchar(75) DEFAULT NULL,
  `firstName` varchar(30) DEFAULT NULL,
  `lastName` varchar(30) DEFAULT NULL,
  `address` varchar(50) DEFAULT NULL,
  `city` varchar(50) DEFAULT NULL,
  `state` varchar(50) DEFAULT NULL,
  `zip` varchar(10) DEFAULT NULL,
  `country` varchar(30) NOT NULL DEFAULT '',
  `phone` varchar(15) DEFAULT NULL,
  `fax` varchar(15) DEFAULT NULL,
  `email` varchar(100) DEFAULT NULL,
  `signupDate` date NOT NULL,
  `retiredOn` date DEFAULT NULL,
  `lastModifiedDate` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `Organization_signupDate` (`signupDate`),
  KEY `Organization_retiredOn` (`retiredOn`)
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8;

CREATE TABLE `Discussion` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(120) DEFAULT NULL,
  `isPublic` tinyint(1) unsigned NOT NULL DEFAULT 0,
  `createdByID` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  KEY `Discussion_createdByID` (`createdByID`),
  KEY `Discussion_isPublic` (`isPublic`)
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8;

CREATE TABLE `DiscussionMessages` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `discussionID` int(10) unsigned NOT NULL,
  `userID` int(10) unsigned NOT NULL,
  `content` mediumtext NOT NULL,
  `timestamp` timestamp NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `DiscussionMessages_timestamp` (`timestamp`),
  CONSTRAINT `fk_DiscussionMessagess_Discussion` FOREIGN KEY (`discussionID`) REFERENCES `Discussion` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_DiscussionMessages_User` FOREIGN KEY (`userID`) REFERENCES `User` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=latin1;

CREATE TABLE `PushSubscriptions` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `userID` int(10) unsigned NOT NULL,
  `type` int(2) unsigned NOT NULL DEFAULT 1,
  `endpoint` text NOT NULL,
  `p256dh` varchar(120) NOT NULL,
  `auth` varchar(120) NOT NULL,
  `expirationTime` datetime,
  `createdOn` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `PushSubscriptions_expirationTime` (`expirationTime`),
  CONSTRAINT `fk_PushSubscriptions_User` FOREIGN KEY (`userID`) REFERENCES `User` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8;

CREATE TABLE `ApplePushSubscriptions` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `userID` int(10) unsigned NOT NULL,
  `type` int(2) unsigned NOT NULL DEFAULT 1,
  `deviceToken` varchar(255) NOT NULL,
  `expirationTime` datetime,
  `createdOn` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `ApplePushSubscriptions_expirationTime` (`expirationTime`),
  CONSTRAINT `fk_ApplePushSubscriptions_User` FOREIGN KEY (`userID`) REFERENCES `User` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8;

CREATE TABLE `FileParts` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `userID` int(10) unsigned NOT NULL,
  `name` varchar(255) DEFAULT NULL,
  `type` varchar(120) NOT NULL,
  `uuid` varchar(120) NOT NULL,
  `etag` varchar(120) NOT NULL,
  `start` int(10) unsigned NOT NULL,
  `size` int(10) unsigned NOT NULL,
  `totalSize` bigint(10) unsigned NOT NULL,
  `createdOn` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `Files_uuid` (`uuid`),
  KEY `Files_createdOn` (`createdOn`),
  CONSTRAINT `fk_Files_User` FOREIGN KEY (`userID`) REFERENCES `User` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8;

CREATE TABLE `LinkUserToDiscussion` (
  `userID` int(10) unsigned NOT NULL,
  `discussionID` int(10) unsigned NOT NULL,
  PRIMARY KEY (`userID`,`discussionID`),
  CONSTRAINT `fk_LinkUserToDiscussion_userID` FOREIGN KEY (`userID`) REFERENCES `User` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_LinkUserToDiscussion_discussionID` FOREIGN KEY (`discussionID`) REFERENCES `Discussion` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `LinkUserToOrg` (
  `userID` int(10) unsigned NOT NULL,
  `orgID` int(10) unsigned NOT NULL,
  `owner` tinyint(1) unsigned DEFAULT 0,
  `apiAccess` tinyint(1) unsigned NOT NULL DEFAULT 0,
  `hidden` tinyint(1) unsigned DEFAULT 0,
  PRIMARY KEY (`userID`,`orgID`),
  KEY `LinkOrgToUser_hidden` (`hidden`),
  CONSTRAINT `fk_LinkUserToOrg_User` FOREIGN KEY (`userID`) REFERENCES `User` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_LinkUserToOrg_Org` FOREIGN KEY (`orgID`) REFERENCES `Organization` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

################################################################################################

CREATE TABLE `LinkDiscussionToOrg` (
  `discussionID` int(10) unsigned NOT NULL,
  `orgID` int(10) unsigned NOT NULL,
  PRIMARY KEY (`discussionID`,`orgID`),
  CONSTRAINT `fk_LinkDiscussionToOrg_Discussion` FOREIGN KEY (`discussionID`) REFERENCES `Discussion` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_LinkDiscussionToOrg_Org` FOREIGN KEY (`orgID`) REFERENCES `Organization` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `UserGroup` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(80) DEFAULT NULL,
  `lastModifiedDate` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8;

CREATE TABLE `OrgGroup` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `type` int(3) unsigned NOT NULL,
  `name` varchar(120) DEFAULT NULL,
  `orgID` int(10) unsigned NOT NULL,
  `lastModifiedDate` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  CONSTRAINT `fk_OrgGroup_Org` FOREIGN KEY (`orgID`) REFERENCES `Organization` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8;

CREATE TABLE `Department` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(120) DEFAULT NULL,
  `orgGroupID` int(10) unsigned NOT NULL,
  `leaderID` int(10) unsigned NOT NULL,
  `lastModifiedDate` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  CONSTRAINT `fk_Department_OrgGroup` FOREIGN KEY (`orgGroupID`) REFERENCES `OrgGroup` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_Department_User` FOREIGN KEY (`leaderID`) REFERENCES `User` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8;

CREATE TABLE `Board` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(120) DEFAULT NULL,
  `isPublic` tinyint(1) unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8;

CREATE TABLE `Blog` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(120) DEFAULT NULL,
  `topic` varchar(255) DEFAULT NULL,
  `body` text DEFAULT NULL,
  `isPublic` tinyint(1) unsigned NOT NULL DEFAULT 0,
  `boardID` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `fk_Blog_Board` FOREIGN KEY (`boardID`) REFERENCES `Board` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8;

CREATE TABLE `Task` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `type` int(3) unsigned NOT NULL,
  `priority` varchar(1) NOT NULL DEFAULT 'A',
  `rank` int(2) NOT NULL DEFAULT 1,
  `status` int(2) unsigned NOT NULL,
  `summary` varchar(255) NOT NULL,
  `description` text DEFAULT NULL,
  `boardID` int(10) unsigned NOT NULL,
  `createDate` datetime NOT NULL,
  `createdByID` int(10) unsigned NOT NULL DEFAULT 0,
  `assigneeID` int(10) unsigned NOT NULL,
  `dueDate` date DEFAULT NULL,
  `estimatedTime` int(6) unsigned NOT NULL DEFAULT 0 COMMENT 'time in min',
  `actualTime` int(6) unsigned NOT NULL DEFAULT 0 COMMENT 'time in min',
  `lastModifiedDate` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `Task_createdByID` (`createdByID`),
  KEY `Task_assigneeID` (`assigneeID`),
  KEY `Task_status` (`status`),
  KEY `Task_type` (`type`),
  CONSTRAINT `fk_Task_Assignee` FOREIGN KEY (`assigneeID`) REFERENCES `User` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_Task_Creator` FOREIGN KEY (`createdByID`) REFERENCES `User` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=latin1;

CREATE TABLE `LinkBlogToUserGroup` (
  `blogID` int(10) unsigned NOT NULL,
  `userGroupID` int(10) unsigned NOT NULL,
  PRIMARY KEY (`blogID`,`userGroupID`),
  CONSTRAINT `fk_LinkBlogToUserGroup_User` FOREIGN KEY (`blogID`) REFERENCES `Blog` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_LinkBlogToUserGroup_UserGroup` FOREIGN KEY (`userGroupID`) REFERENCES `UserGroup` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `zSystemType` (
  `id` int(2) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(45) NOT NULL,
  `description` varchar(120) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT INTO `zSystemType` (`id`, `name`, `description`)
VALUES
	(1, 'household', 'Household'),
	(2, 'admin', 'Administration'),
	(3, 'hr', 'Human Resources'),
	(4, 'customer', 'Customer'),
	(5, 'marketing', 'Marketing'),
	(6, 'fulfillment', 'Fulfillment'),
	(7, 'finance', 'Finance');
