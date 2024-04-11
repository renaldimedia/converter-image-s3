-- converters3.converted_files definition

CREATE TABLE `converted_files` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `filename` text NOT NULL,
  `size` bigint(20) unsigned DEFAULT NULL,
  `converted_time` datetime NOT NULL DEFAULT current_timestamp(),
  `filepath` text DEFAULT NULL,
  `endpoint` varchar(255) NOT NULL,
  `bucket` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=5535 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;