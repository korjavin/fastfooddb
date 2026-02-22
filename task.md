
Open Food Facts Local Mirror Service in Go
Task and architecture doc
Goal
Build a small Go service that serves two queries fast from local storage.
Exact barcode lookup returns name plus macros plus kcal
Name search returns top matches with barcode plus name plus macros plus kcal
Works with a few GB RAM by relying on disk plus OS page cache
Import runs offline as a CLI once, then you copy the produced data directory to prod
Simple API key auth
Non goals
Full Open Food Facts schema
Real time sync with upstream
Perfect ranking for every language
Guarantee microseconds for fuzzy search. Barcode can be microseconds. Name search will be milliseconds on SSD, which is still great for UI.
Proposed stack
Go service
Pebble embedded KV for barcode to payload
Bleve embedded full text index for name search
Optional gzip JSONL dump as input for importer
Why this combo
Pebble gives very fast exact gets by key
Bleve gives flexible search including fuzziness and tokenization
High level architecture
Data build time
Dump file → Importer CLI → data directory
data directory contains
pebble store
bleve index
manifest json with build info
Runtime
Client → HTTP JSON API → auth middleware → query handlers
Handlers use
Bleve for search to get doc ids
Pebble for final payload fetch
Diagram
Dump gzip JSONL
→ importer CLI
→ data dir
→ pebble
→ bleve
→ manifest
Copy data dir to prod
→ service opens pebble and bleve read only
→ serves requests
Data model
Minimal payload we store
Per product
barcode string
name string
nutriments
kcal per 100g and optional per serving
protein g per 100g
fat g per 100g
carbs g per 100g

optional metadata
lang used for name
last modified timestamp from dump if available

Pebble keys and values
Key
barcode as bytes
Value
compact binary record, versioned
Suggested binary layout
version uvarint
name length uvarint then utf8 bytes
kcal100g float32
protein100g float32
fat100g float32
carbs100g float32
kcalServing float32 optional, or store as NaN if missing
salt fields later as reserved
Reason
fast decode, small, stable, no json overhead
Bleve documents
Document id
barcode
Indexed fields
name_folded
name_raw optional for debugging
Stored fields
none, keep index small, final data comes from Pebble
Name folding function used both at index and query time
lower case
unicode normalize then strip combining marks
map ß to ss
collapse spaces
keep letters and digits, turn other chars into spaces
Search behavior
Query types
Exact barcode
just pebble get
Name search
two stage query in Bleve
Stage A
exact phrase or match query boosted
prefix query boosted for typeahead feel
Stage B
fuzzy query edit distance 1 for each token
Optional edit distance 2 only when token length is large, for example 8 plus
Return top N doc ids, then batch get from Pebble
Ranking
Use Bleve scoring out of the box at first
Add manual rerank later if needed using
edit distance between folded query and folded name
token overlap
Result shape
For search return an array of products with
barcode, name, kcal100g, protein100g, fat100g, carbs100g
Import pipeline
Input
Open Food Facts products dump in JSONL gzip
Streaming parse
open gzip reader
scan line by line
json unmarshal into a minimal struct with only needed fields
extract barcode and name
extract nutriments
Field extraction rules
Barcode
use code field, skip if empty
Name
prefer product_name in requested language if present
fallback order
product_name
product_name_en
generic_name
short_name
last resort empty, skip indexing but still store if barcode exists
Nutriments
prefer per 100g fields
energy kcal per 100g if present
else compute from energy kj per 100g if present
Macros per 100g
proteins, fat, carbohydrates
If missing keep as NaN
Write strategy
open new pebble dir
open new bleve index dir
for each valid product
write pebble record by barcode
index folded name into bleve with doc id barcode

flush and close
write manifest json with
build time, dump source, counts, schema version, git sha
final output is a single directory ready to copy
Idempotency
Importer always writes into a fresh output directory
No in place updates
Runtime service
API endpoints
GET /v1/product/{barcode}
returns one payload or 404
GET /v1/search?q=spaghetti&limit=10
returns list of payloads
GET /health
returns ok plus manifest version
Authentication
API key required for all endpoints except health
Option A simplest
Use Authorization header
Authorization: ApiKey YOURKEY
Server config has a list of allowed keys
Option B safer storage
Store only sha256 hashes of keys in config
At request
sha256 of provided key then constant time compare
Go notes
use crypto subtle constant time compare
do not log raw keys
Rate limiting
Optional but recommended
token bucket per key in memory
Protects against fuzzy search abuse
Concurrency
Pebble reads are safe concurrently
Bleve search is safe concurrently
Use a small worker pool if you want to cap CPU
Config
Environment variables or config file in data dir
DATA_DIR
PORT
API_KEYS or API_KEY_HASHES
DEFAULT_LIMIT
MAX_LIMIT
Deployment model
Build once then copy
Run importer on a machine with enough disk
Copy the produced data directory to prod server
Start service pointing at that directory
Zero downtime swap
Keep two data dirs
data_current and data_next
copy new data into data_next
restart service with DATA_DIR set to data_next
or do an atomic symlink swap and restart
Backups
Treat data dir as an artifact
Store it in object storage with manifest
Performance targets and measurement
Targets
Barcode lookup
median under 200 microseconds inside handler when warmed
p99 under 2 ms
Name search
median under 5 ms warmed
p99 under 30 ms for typical queries
Measurement plan
add internal timings in logs
add p50 p95 p99 metrics per endpoint
run a local load test with realistic query set
test cold cache and warm cache separately
Tasks
Milestone 1 minimal prototype
Repo skeleton
Data structs for minimal OFF fields
CLI importer reads a small sample dump and builds pebble plus bleve
HTTP server with two endpoints and basic json responses
Simple API key middleware
Acceptance
You can import sample and run search and barcode locally
Milestone 2 production grade importer
Streaming importer for full dump
Robust folding and tokenization
Manifest file
Progress reporting and counters
Validation rules and skip logs
Build reproducibility controls
same input gives same counts
Acceptance
Full dump import completes without OOM
Output dir size is acceptable
Spot checks match upstream values
Milestone 3 performance and correctness
Batch gets from pebble for search results
Tune bleve mapping
Add query blend boosts and fuzzy distance policy
Add benchmarks for
barcode get
search with common terms
Add p99 metrics
Acceptance
Meets latency targets on prod like hardware
Milestone 4 ops and security
API key hashes option
Rate limiting per key
Structured logging
Health endpoint returns manifest version
Dockerfile and systemd unit example
Documentation for build and deploy
Acceptance
Simple deploy and safe key rotation
Risks and mitigations
Bleve index size grows too much
Mitigation
index only folded name, store nothing, keep analyzers simple
Fuzzy search too slow on very short queries
Mitigation
disable fuzzy for tokens shorter than 4
prefer prefix and exact match
Multi language names cause noise
Mitigation
index only one chosen name field at first
add extra language fields later if needed
Data license and attribution requirements
Mitigation
add a LICENSE and attribution endpoint
document how the data is sourced and rebuilt
Deliverables
offimport CLI
offserve HTTP service
data artifact format
runbook for import and deploy
benchmarks and load test script
If you want, I can also give you a concrete folder layout and Go package structure, plus the exact bleve index mapping and a small example of the importer command interface without any flags.