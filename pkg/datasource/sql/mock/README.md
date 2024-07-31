```bash
mockgen -source=test_driver.go -destination=./mock_driver.go -package=mock
mockgen -source=../datasource/datasource_manager.go -destination=./mock_datasource_manager.go -package=mock
```