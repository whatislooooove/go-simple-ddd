# Документация блокчейна

## Содержание
1. [Обзор](#обзор)
2. [Архитектура](#архитектура)
3. [Файлы и их назначение](#файлы-и-их-назначение)
4. [История коммитов](#история-коммитов)
5. [Как это работает](#как-это-работает)
6. [TODO и проблемы](#todo-и-проблемы)
7. [Что нужно сделать](#что-нужно-сделать)

---

## Обзор

Проект реализует упрощённый блокчейн на Go с поддержкой:
- Proof of Work (PoW)
- Транзакций с подписью ECDSA
- Системы кошельков (адреса в формате Base58)
- Хранения в BadgerDB

---

## Архитектура

```
cmd/blockchain/main.go          # CLI интерфейс
internal/domain/
├── block/
│   ├── block.go                # Структура блока, сериализация
│   └── proof.go                # Proof of Work
├── chain/
│   ├── chain.go                # Логика блокчейна, транзакции, UTXO
│   └── chain_iterator.go       # Итератор по блокам
├── transaction/
│   └── transaction.go          # Структуры транзакций, подпись, верификация
├── wallet/
│   ├── wallet.go               # Кошелёк, генерация ключей, адреса
│   ├── wallets.go              # Менеджер кошельков (файловое хранение)
│   └── utils.go                # Base58 encode/decode
└── shared/
    └── errors.go               # Общие ошибки
```

---

## Файлы и их назначение

### `cmd/blockchain/main.go`
CLI-утилита для взаимодействия с блокчейном.

**Команды:**
- `createblockchain -address {address}` — создаёт генезис-блок с coinbase-транзакцией на указанный адрес
- `printchain` — выводит все блоки
- `getbalance -address {address}` — показывает баланс адреса
- `send -from {from} -to {to} -amount {amount}` — отправляет транзакцию
- `createwallet` — создаёт новый кошелёк
- `listaddresses` — выводит все адреса из файла кошельков


### `internal/domain/block/block.go`
Структура блока и методы сериализации.

```go
type Block struct {
    Hash         []byte
    Transactions []*transaction.Transaction
    PrevHash     []byte
    Nonce        int
}
```

- `CreateBlock()` — создаёт блок с PoW
- `CreateGenesisBlock()` — создаёт генезис-блок
- `Serialize()` / `Deserialize()` — gob-сериализация для BadgerDB

### `internal/domain/block/proof.go`
Алгоритм Proof of Work.

- `Difficulty = 10` — количество ведущих нулей в хэше
- `Run()` — перебирает nonce, пока хэш не будет меньше target
- `Validate()` — проверяет валидность блока

**Проблема:** `Run()` выводит каждый хэш через `fmt.Printf`, что создаёт огромный вывод при майнинге. Есть TODO про вывод в одну строку.

### `internal/domain/chain/chain.go`
Основная логика блокчейна.

**Структура:**
```go
type BlockChain struct {
    LastHash []byte
    DataBase *badger.DB
}
```

**Ключевые методы:**
- `InitBlockChain(address)` — создаёт новый блокчейн с генезис-блоком
- `ContinueBlockChain(address)` — открывает существующий блокчейн
- `AddBlock(transactions)` — добавляет блок с транзакциями
- `FindUnspentTransactions(pubKeyHash)` — ищет непотраченные транзакции (UTXO)
- `FindUTXO(pubKeyHash)` — возвращает все UTXO для адреса
- `FindSpendableOutputs(pubKeyHash, amount)` — находит достаточно средств для суммы
- `NewTransaction(from, to, amount, chain)` — создаёт подписанную транзакцию
- `SignTransaction()` — подписывает транзакцию приватным ключом
- `VerifyTransaction()` — проверяет подпись транзакции

### `internal/domain/chain/chain_iterator.go`
Итератор для обхода блоков от последнего к генезису.

### `internal/domain/transaction/transaction.go`
Структуры и логика транзакций.

```go
type Transaction struct {
    ID      []byte
    Inputs  []TxInput
    Outputs []TxOutput
}

type TxInput struct {
    ID        []byte
    Out       int
    Signature []byte
    PubKey    []byte
}

type TxOutput struct {
    Value      int
    PubKeyHash []byte
}
```

**Ключевые методы:**
- `SetID()` — вычисляет хэш транзакции (устаревший, используется `Hash()`)
- `Hash()` — вычисляет хэш trimmed-копии транзакции
- `IsCoinbase()` — проверяет, является ли транзакция coinbase
- `Sign()` — подписывает все входы
- `Verify()` — проверяет подпись всех входов
- `TrimmedCopy()` — создаёт копию без подписей для хэширования


### `internal/domain/wallet/wallet.go`
Работа с криптографическими ключами и адресами.

```go
type Wallet struct {
    PrivateKey ecdsa.PrivateKey
    PublicKey  []byte
}
```

**Ключевые методы:**
- `NewKeyPair()` — генерирует пару ключей ECDSA (P-256)
- `MakeAddress()` — создаёт адрес из публичного ключа (SHA256 → RIPEMD160 → Base58)
- `ValidateAddress()` — проверяет валидность адреса (checksum)
- `PublicKeyHash()` — SHA256 + RIPEMD160
- `GenerateChecksum()` — двойной SHA256, первые 4 байта

**Сериализация:**
- `MarshalJSON()` / `UnmarshalJSON()` — кастомная JSON-сериализация (решает проблему с `elliptic.Curve`)

### `internal/domain/wallet/wallets.go`
Менеджер кошельков с файловым хранением.

```go
type Wallets struct {
    Wallets map[string]*Wallet  // address -> Wallet
}
```

- `SaveFile()` / `LoadFile()` — JSON в `./tmp/wallet.json`
- `AddWallet()` — создаёт новый кошелёк и добавляет в мапу
- `GetWallet(address)` — возвращает кошелёк по адресу

### `internal/domain/wallet/utils.go`
Утилиты для Base58 кодирования/декодирования адресов.

---

## Как это работает

### 1. Создание блокчейна
```
createblockchain -address {address}
```
1. Создаётся coinbase-транзакция на 100 монет для `address`
2. Создаётся генезис-блок с этой транзакцией
3. Блок сохраняется в BadgerDB
4. В базу записывается `last-hash`

### 2. Создание кошелька
```
createwallet
```
1. Генерируется пара ключей ECDSA P-256
2. Из публичного ключа создаётся адрес (SHA256 → RIPEMD160 → Base58 с checksum)
3. Кошелёк сохраняется в `tmp/wallet.json`

### 3. Проверка баланса
```
getbalance -address {address}
```
1. Валидируется адрес (checksum)
2. Декодируется адрес, извлекается pubKeyHash
3. Ищутся все UTXO для pubKeyHash
4. Суммируются значения выходов

### 4. Отправка транзакции
```
send -from {from} -to {to} -amount {amount}
```
1. Загружаются кошельки, находится кошелёк отправителя
2. Вычисляется pubKeyHash отправителя
3. Ищутся тратимые выходы (`FindSpendableOutputs`)
4. Если средств недостаточно — паника "Not enough funds"
5. Создаются входы из найденных UTXO
6. Создаётся выход на получателя
7. Если есть сдача — создаётся выход на отправителя
8. Транзакция подписывается приватным ключом отправителя
9. Блок с транзакцией добавляется в цепь

### 5. Proof of Work
При создании блока:
1. Берётся `PrevHash` + хэш всех транзакций + nonce + difficulty
2. Вычисляется SHA256
3. Если хэш >= target — nonce++
4. Если хэш < target — блок найден

---

## TODO и проблемы

### Критические баги

1. **`NewTransaction` не проверяет, что `w != nil`** ([chain.go:211](internal/domain/chain/chain.go:211))
   Если адрес не найден в кошельках, будет panic при обращении к `w.PublicKey`.

### Проблемы с сериализацией

2. **JSON-сериализация `ecdsa.PrivateKey`** — уже частично решено через `MarshalJSON`/`UnmarshalJSON`, но:
   - `WalletJSON` экспортирован (начинается с большой буквы), хотя используется только внутри пакета
   - Нет валидации при десериализации (например, проверки что приватный ключ валиден для кривой)

3. **`SetID()` устарел** ([transaction.go:37](internal/domain/transaction/transaction.go:37))
   В `NewTransaction` используется `tx.Hash()`, но `CoinbaseTx` всё ещё вызывает `tx.SetID()`. Нужно унифицировать.

### Архитектурные проблемы

4. **Жёстко прописанные пути** ([wallets.go:12](internal/domain/wallet/wallets.go:12), [chain.go:20](internal/domain/chain/chain.go:20))
   ```go
   const walletFile = "./tmp/wallet.json"
   const dbPath = "./tmp/chain.db"
   ```
   Должны быть конфигурируемыми.

5. **`Wallets` хранится в памяти** ([wallets.go:14](internal/domain/wallet/wallets.go:14))
   При каждом вызове `CreateWallets()` файл читается заново. Нет кэширования.
6. **Нет валидации подписей при добавлении блока** ([chain.go:30](internal/domain/chain/chain.go:30))
    `AddBlock` не вызывает `VerifyTransaction`. Любая подпись принимается.

### TODO из кода
7. **`version = byte(0x00)`** ([wallet.go:20](internal/domain/wallet/wallet.go:20)) — "todo выяснить почему так"
    Версия адреса всегда 0x00 (как в Bitcoin). Нужно документировать.
8. **`pub := append(private.PublicKey.X.Bytes(), ...)`** ([wallet.go:100](internal/domain/wallet/wallet.go:100)) — "todo обновить"
    Публичный ключ должен быть в несжатом формате (65 байт: 0x04 + X + Y). Сейчас просто конкатенируются байты X и Y без префикса 0x04.

9. **`address длиннее чем pubHash`** ([wallet.go:75](internal/domain/wallet/wallet.go:75)) — "todo:почему то адрес длиннее чем pubHash"
    Base58 кодирует больше байт (версия + pubHash + checksum), поэтому адрес длиннее. Это нормально.

10. **`Base58Encode` возвращает `[]byte`** ([utils.go:9](internal/domain/wallet/utils.go:9)) — "todo тоже выяснить как работает"
    Base58 конвертирует байты в строку, затем возвращается как `[]byte`. Это неэффективно.

11. **`Wallets` файл в другом месте** ([wallets.go:10](internal/domain/wallet/wallets.go:10)) — "todo: сделать рефакторинг и поместить файл в другом месте. Сделать прослойку в виде репозитория"
    Нужен интерфейс `WalletRepository` с методами `Save`, `Load`, `GetAll`.

12. **`run()` стоит вынести** ([main.go:37](internal/domain/blockchain/main.go:37)) — "todo тоже стоит куда то вынести"
    Логика CLI должна быть в отдельном пакете/сервисе.

13. **`getBalance` проверка** ([main.go:152](internal/domain/blockchain/main.go:152)) — "TODO:вынести проверку"
    Валидация адреса должна быть в `wallet` пакете или в сервисе.

14. **`send` отправка токенов** ([main.go:173](internal/domain/blockchain/main.go:173)) — "TODO: исправить отправку токенов"
    См. баг #1 с инвертированной логикой.

15. **`Difficulty = 10`** ([proof.go:13](internal/domain/block/proof.go:13))
    Сложность захардкожена. Должна быть конфигурируемой или адаптивной.

16. **`fmt.Printf` в `Run()`** ([proof.go:48](internal/domain/block/proof.go:48))
    Выводит каждый хэш при майнинге. Должно быть опционально или убрано.

---

## Что нужно сделать

### Срочно (критические баги)
- [ ] Исправить инвертированную логику в `Sign()` — `!= nil` → `== nil`
- [ ] Исправить `ContinueBlockChain()` — `DBExists() == false` → `DBExists() == true`
- [ ] Добавить проверку `w != nil` в `NewTransaction()`
- [ ] Добавить валидацию подписей в `AddBlock()`

### Важно (архитектура)
- [ ] Убрать `block.HandleError`, использовать только `shared.HandleError`
- [ ] Сделать `WalletRepository` интерфейс для persistence
- [ ] Вынести CLI-логику из `main.go` в отдельный пакет
- [ ] Сделать пути к БД и файлу кошельков конфигурируемыми
- [ ] Добавить кэширование кошельков в `Wallets`
