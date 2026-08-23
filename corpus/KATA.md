# bigo kata corpus — oracle golden

GENERATED — do not edit; regenerate with `task kata-corpus`.

Human-claimed bounds on real submitted solutions, scored under the KATA cost
model. `exact` = inference matches the author's claim; `loose` = sound but
imprecise, which here often means bigo answered the worst case where the
author claimed the average one; `top` = unverifiable. A `wrong` never appears:
it fails the build. What was reduced away, and why, is in
[kata/README.md](kata/README.md). **This is not a coverage metric** — read
composition, not a percentage.

Sibling: [CORPUS.md](CORPUS.md) pins literature bounds under the default cost
model. The two are never summed.

**Entries: 31**

## Time statuses

| Status | Count |
|---|---|
| exact | 19 |
| loose | 0 |
| top | 12 |

## Space statuses (pinned entries only)

| Status | Count |
|---|---|
| exact | 20 |
| loose | 0 |
| top | 11 |

## Per family

| Family | Entries |
|---|---|
| brokensearch | 1 |
| deque | 10 |
| hashtable | 7 |
| quicksort | 3 |
| rpncalc | 5 |
| searchengine | 5 |

## Entries

| Function | Time pin | Time got | Status | Space pin | Space got | Status | Cause | Source |
|---|---|---|---|---|---|---|---|---|
| brokensearch.BrokenSearch | O(log(len(arr))) | O(log(len(arr))) | exact | O(1) | O(1) | exact |  | ya_algo sprint 3 final 1; author's claim "В решении используется только бинарный поиск, который работает за O(log n)", auxiliary space "O(1) - можно сказать, что почти не используется" |
| deque.CalculatePosition | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 1; one modulo, constant by inspection |
| deque.IsEmpty | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 1; author's claim "Выполнение одной команды - O(1)" |
| deque.IsFull | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 1; author's claim "Выполнение одной команды - O(1)" |
| deque.MoveHead | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 1; index arithmetic, constant by inspection |
| deque.MoveTail | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 1; index arithmetic, constant by inspection |
| deque.NewDeque | O(1) | O(1) | exact | O(n) | O(n) | exact |  | ya_algo sprint 2 final 1; author's claim "Ииницализация слайса для хранения буфера - О(k), где k - размер буфера", constant work beyond the allocation |
| deque.PopBack | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 1; author's claim "Выполнение одной команды - O(1)" |
| deque.PopFront | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 1; author's claim "Выполнение одной команды - O(1)" |
| deque.PushBack | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 1; author's claim "Выполнение одной команды - O(1)" |
| deque.PushFront | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 1; author's claim "Выполнение одной команды - O(1)" |
| hashtable.Delete | O(ht.capacity) | unverifiable | top | O(1) | unverifiable | top | call | ya_algo sprint 4 final 2; author's worst case O(N) for the chain walk |
| hashtable.DeleteNode | O(ht.capacity) | unverifiable | top | O(1) | O(1) | exact | call | ya_algo sprint 4 final 2; same chain walk as FindNode, which it calls |
| hashtable.FindNode | O(ht.capacity) | unverifiable | top | O(1) | O(1) | exact | loop | ya_algo sprint 4 final 2; author's worst case "когда вообще все значения попадут в одну ячейку таблицы ХТ, сложность операций будет O(N)" — the chain walk is the N term. Pinned against capacity as the only size in scope; the element count N is not a size bigo can name here. |
| hashtable.Get | O(ht.capacity) | unverifiable | top | O(1) | unverifiable | top | call | ya_algo sprint 4 final 2; author's worst case O(N) for the chain walk |
| hashtable.Index | O(1) | unverifiable | top | O(1) | unverifiable | top | call | ya_algo sprint 4 final 2; author's claim "Вычисление хеша ключа - O(1)" and "Вычисление номера ячейки таблицы ХТ - O(1)" |
| hashtable.NewHashTable | O(1) | O(1) | exact | O(hashTableCapacity) | unverifiable | top |  | ya_algo sprint 4 final 2; the table slice is allocated at capacity M whether or not anything is stored |
| hashtable.Put | O(ht.capacity) | unverifiable | top | O(1) | unverifiable | top | call | ya_algo sprint 4 final 2; author's average case "O(1+N/M) или O(1+α)", worst case O(N) when every key lands in one bucket. The worst case is pinned, since that is the question bigo answers. |
| quicksort.Less | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 3 final 2; the author's claim states "сравнение структур-участников - это O(1)" — one participant comparison is one element operation under the kata cost model |
| quicksort.Partition | O(len(participants)) | unverifiable | top | O(1) | unverifiable | top | loop | ya_algo sprint 3 final 2; the author's claim counts partition as the O(n) per-level element work inside quicksort's O(n log n) |
| quicksort.QuickSort | O(len(participants) log(len(participants))) | unverifiable | top | O(log(len(participants))) | unverifiable | top | call | ya_algo sprint 3 final 2; author's claim "решение работает за O(n * log n), свойственное среднему случаю быстрой сортировки", auxiliary space "O(log n) дополнительной памяти" for the recursion stack |
| rpncalc.Calculate | O(len(c.exp)) | unverifiable | top | O(len(c.exp)) | unverifiable | top | loop | ya_algo sprint 2 final 2; author's claim "Общее время - O(n+n) - O(2n) - O(n)", where n is the token count. Pinned against len(c.exp) because the token count is not a size bigo can name; tokens ≤ characters, so the pin is sound in the direction that matters. |
| rpncalc.NewCalculator | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 2; struct construction, constant by inspection |
| rpncalc.NewStack | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 2; allocation of an empty slice, constant by inspection |
| rpncalc.Pop | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 2; author's claim "извлечение двух операндов из стека - O(1)" |
| rpncalc.Push | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 2; author's claim "На каждый токен-операнд выполняется помещение в стек - O(1)". Amortized append rides bigo's documented primitive licence. |
| searchengine.GetUniqueWords | O(len(query)) | unverifiable | top | O(len(query)) | unverifiable | top | loop | ya_algo sprint 4 final 1; author's claim "Обработка входщих запросов, получение уникальных слов - O(M)". Pinned against len(query): words ≤ characters. |
| searchengine.Less | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 4 final 1; comparator, constant work per call under the kata cost model |
| searchengine.NewSearchEngine | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 4 final 1; map construction, constant by inspection |
| searchengine.ProcessQuery | O(len(query)^2) | unverifiable | top | O(len(query)) | unverifiable | top | call | ya_algo sprint 4 final 1; author's claim "O(N + M + K*L)" for the whole solution, of which this function is the M + K*L part. Pinned against len(query) as the only size bigo can name here: both the unique-word count K and the candidate-document count are bounded by the query length in this fixture's terms, and the squared term covers the limited bubble sort. |
| searchengine.UpdateIndex | O(len(doc)) | unverifiable | top | O(len(doc)) | unverifiable | top | loop | ya_algo sprint 4 final 1; author's claim "Построение поискового индекса - O(N)" over the words of the documents. Pinned against len(doc) because the word count is not a size bigo can name; words ≤ characters, so the pin is sound in the direction that matters. |
