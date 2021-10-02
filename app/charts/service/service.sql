SELECT * FROM service as service
JOIN JSON_TABLE(service.price_options, '$[*]' columns (price varchar(50) path '$') ) as t group by price

/**
[mysqld]
sql_mode = "STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION"
 */
 