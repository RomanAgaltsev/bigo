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

**Entries: 61**

## Time statuses

| Status | Count |
|---|---|
| exact | 39 |
| loose | 0 |
| top | 22 |

## Space statuses (pinned entries only)

| Status | Count |
|---|---|
| exact | 44 |
| loose | 0 |
| top | 17 |

## Per family

| Family | Entries |
|---|---|
| brokensearch | 1 |
| cheatsheet | 3 |
| deque | 10 |
| expensivenetwork | 7 |
| hashtable | 7 |
| heapsort | 9 |
| levenshtein | 1 |
| packedprefix | 4 |
| quicksort | 3 |
| railroads | 4 |
| removenode | 2 |
| rpncalc | 5 |
| searchengine | 5 |

## Entries

| Function | Time pin | Time got | Status | Space pin | Space got | Status | Cause | Source |
|---|---|---|---|---|---|---|---|---|
| brokensearch.BrokenSearch | O(log(len(arr))) | O(log(len(arr))) | exact | O(1) | O(1) | exact |  | ya_algo sprint 3 final 1; author's claim "В решении используется только бинарный поиск, который работает за O(log n)", auxiliary space "O(1) - можно сказать, что почти не используется" |
| cheatsheet.Insert | O(len(word)) | O(len(word)) | exact | O(len(word)) | O(len(word)) | exact |  | ya_algo sprint 8 final 2; author's claim "Сложность формирования префиксного дерева составляет O(L)" over the total length of all words — one word contributes its own length |
| cheatsheet.IsLineSplitable | O(len(line)^2) | O(len(line)^2) | exact | O(len(line)) | O(len(line)) | exact |  | ya_algo sprint 8 final 2; author's claim "Проверка возможности разбивки строки составляет O(T * M)", T the line length and M the longest word. M is not a size this function's parameters can name, so the pin states the author's own bound with M <= T substituted — a word longer than the line can never match, so the substitution is sound and does not weaken the claim. |
| cheatsheet.NewNode | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 8 final 2; map construction, constant by inspection |
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
| expensivenetwork.AddEdge | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 6 final 1; two single-element appends, riding the amortization licence |
| expensivenetwork.Len | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 6 final 1; one len, constant by inspection |
| expensivenetwork.Less | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 6 final 1; one comparison, constant by inspection |
| expensivenetwork.Max | O(g.edgesCount log(g.edgesCount)) | unverifiable | top | O(g.edgesCount) | unverifiable | top | call | ya_algo sprint 6 final 1; author's claim "Оценка для алгоритма Прима - O(E*log(V))". Pinned against the edge count, the only one of E and V this function names as a size; every edge enters the queue at most once and each queue operation is logarithmic. |
| expensivenetwork.Pop | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 6 final 1; a reslice and an index, constant by inspection |
| expensivenetwork.Push | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 6 final 1; one single-element append, riding the amortization licence |
| expensivenetwork.Swap | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 6 final 1; one exchange, constant by inspection |
| hashtable.Delete | O(ht.capacity) | unverifiable | top | O(1) | O(1) | exact | call | ya_algo sprint 4 final 2; author's worst case O(N) for the chain walk |
| hashtable.DeleteNode | O(ht.capacity) | unverifiable | top | O(1) | O(1) | exact | call | ya_algo sprint 4 final 2; same chain walk as FindNode, which it calls |
| hashtable.FindNode | O(ht.capacity) | unverifiable | top | O(1) | O(1) | exact | loop | ya_algo sprint 4 final 2; author's worst case "когда вообще все значения попадут в одну ячейку таблицы ХТ, сложность операций будет O(N)" — the chain walk is the N term. Pinned against capacity as the only size in scope; the element count N is not a size bigo can name here. |
| hashtable.Get | O(ht.capacity) | unverifiable | top | O(1) | O(1) | exact | call | ya_algo sprint 4 final 2; author's worst case O(N) for the chain walk |
| hashtable.Index | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 4 final 2; author's claim "Вычисление хеша ключа - O(1)" and "Вычисление номера ячейки таблицы ХТ - O(1)" |
| hashtable.NewHashTable | O(1) | O(1) | exact | O(hashTableCapacity) | unverifiable | top |  | ya_algo sprint 4 final 2; the table slice is allocated at capacity M whether or not anything is stored |
| hashtable.Put | O(ht.capacity) | unverifiable | top | O(1) | O(1) | exact | call | ya_algo sprint 4 final 2; author's average case "O(1+N/M) или O(1+α)", worst case O(N) when every key lands in one bucket. The worst case is pinned, since that is the question bigo answers. |
| heapsort.Build | O(h.length) | unverifiable | top | O(1) | unverifiable | top | call | ya_algo sprint 5 final 1; author's claim "Построение кучи - O(n) - при построении кучи обрабатываются все элементы исходного массива" |
| heapsort.HeapifyIterative | O(log(h.length)) | unverifiable | top | O(1) | unverifiable | top | loop | ya_algo sprint 5 final 1; author's claim "просеивание ... зависит от высоты кучи, это O(log n)", no recursion stack in this form |
| heapsort.HeapifyRecursive | O(log(h.length)) | unverifiable | top | O(log(h.length)) | unverifiable | top | call | ya_algo sprint 5 final 1; author's claim "перестановка (просеивание) элемент зависит от высоты кучи, это O(log n)". Space is the recursion stack, one frame per level. |
| heapsort.Largest | O(1) | unverifiable | top | O(1) | unverifiable | top | call | ya_algo sprint 5 final 1; two comparisons through the stored comparator, constant per call under the kata cost model |
| heapsort.Left | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 5 final 1; index arithmetic, constant by inspection |
| heapsort.NewHeap | O(len(arr)) | O(len(arr)) | exact | O(len(arr)) | O(len(arr)) | exact |  | ya_algo sprint 5 final 1; the array is copied once into the one-indexed backing slice |
| heapsort.Right | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 5 final 1; index arithmetic, constant by inspection |
| heapsort.Sort | O(h.length log(h.length)) | unverifiable | top | O(log(h.length)) | unverifiable | top | call | ya_algo sprint 5 final 1; author's claim "получаем O(n + n * log n) или O(n * log n)". Space is the sift recursion; the array itself is the input, not auxiliary. |
| heapsort.Swap | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 5 final 1; one exchange, constant by inspection |
| levenshtein.LevenshteinDistance | O(len(s) len(t)) | unverifiable | top | O(len(s)) | unverifiable | top | loop | ya_algo sprint 7 final 1; author's claim "для решения необходимо выполнить M * N операций. Получается оценка O(M * N)", space "создается два слайса размера N+1 ... получаем оценку по памяти - O(N)" |
| packedprefix.Pop | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 8 final 1; a reslice and an index, constant by inspection |
| packedprefix.Push | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 8 final 1; one append of a single element, riding the amortization licence |
| packedprefix.Size | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 8 final 1; one len, constant by inspection |
| packedprefix.Unpack | O(len(packedLine)) | unverifiable | top | O(len(packedLine)) | unverifiable | top | call | ya_algo sprint 8 final 1; author's claim "Функция распаковки работает за O(M) для каждой строки - обрабатывается каждый символ строки", after his own correction from O(n * M^2) to O(n * M) for the whole solution |
| quicksort.Less | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 3 final 2; the author's claim states "сравнение структур-участников - это O(1)" — one participant comparison is one element operation under the kata cost model |
| quicksort.Partition | O(len(participants)) | unverifiable | top | O(1) | unverifiable | top | loop | ya_algo sprint 3 final 2; the author's claim counts partition as the O(n) per-level element work inside quicksort's O(n log n) |
| quicksort.QuickSort | O(len(participants) log(len(participants))) | unverifiable | top | O(log(len(participants))) | unverifiable | top | call | ya_algo sprint 3 final 2; author's claim "решение работает за O(n * log n), свойственное среднему случаю быстрой сортировки", auxiliary space "O(log n) дополнительной памяти" for the recursion stack |
| railroads.AddRoad | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 6 final 2; one append of a single element, riding the amortization licence |
| railroads.MapIsOptimal | O(g.verticesCount) | unverifiable | top | O(g.verticesCount) | unverifiable | top | loop | ya_algo sprint 6 final 2; author's claim "фактически используется DFS, то и сложность алгоритма будет такой же - O(V + E)". Pinned against the vertex count, the only one of V and E this function names as a size. |
| railroads.NewGraph | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 6 final 2; struct construction, constant by inspection |
| railroads.PathHasCycles | O(g.verticesCount) | unverifiable | top | O(g.verticesCount) | unverifiable | top | loop | ya_algo sprint 6 final 2; the DFS visit of the author's O(V + E) claim; space is the recursion stack, at most one frame per vertex |
| removenode.Remove | O(n) | unverifiable | top | O(n) | unverifiable | top | call | ya_algo sprint 5 final 2; author's claim "получаем сложность решения O(h)" for the average case, with O(n) named as the worst case when the tree is one chain. Space is the recursion stack, author's "для хранения стека рекурсивных вызовов требуется O(h)". |
| removenode.Successor | O(n) | unverifiable | top | O(1) | O(1) | exact | loop | ya_algo sprint 5 final 2; author's claim "найти его преемника - O(n) в худшем, O(h) в среднем случае". The worst case is pinned: h = n when the tree degenerates to a chain, and that is the question bigo answers. |
| rpncalc.Calculate | O(len(c.exp)) | unverifiable | top | O(len(c.exp)) | unverifiable | top | call | ya_algo sprint 2 final 2; author's claim "Общее время - O(n+n) - O(2n) - O(n)", where n is the token count. Pinned against len(c.exp) because the token count is not a size bigo can name; tokens ≤ characters, so the pin is sound in the direction that matters. |
| rpncalc.NewCalculator | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 2; struct construction, constant by inspection |
| rpncalc.NewStack | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 2; allocation of an empty slice, constant by inspection |
| rpncalc.Pop | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 2; author's claim "извлечение двух операндов из стека - O(1)" |
| rpncalc.Push | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 2 final 2; author's claim "На каждый токен-операнд выполняется помещение в стек - O(1)". Amortized append rides bigo's documented primitive licence. |
| searchengine.GetUniqueWords | O(len(query)) | unverifiable | top | O(len(query)) | unverifiable | top | loop | ya_algo sprint 4 final 1; author's claim "Обработка входщих запросов, получение уникальных слов - O(M)". Pinned against len(query): words ≤ characters. |
| searchengine.Less | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 4 final 1; comparator, constant work per call under the kata cost model |
| searchengine.NewSearchEngine | O(1) | O(1) | exact | O(1) | O(1) | exact |  | ya_algo sprint 4 final 1; map construction, constant by inspection |
| searchengine.ProcessQuery | O(len(query)^2) | unverifiable | top | O(len(query)) | unverifiable | top | call | ya_algo sprint 4 final 1; author's claim "O(N + M + K*L)" for the whole solution, of which this function is the M + K*L part. Pinned against len(query) as the only size bigo can name here: both the unique-word count K and the candidate-document count are bounded by the query length in this fixture's terms, and the squared term covers the limited bubble sort. |
| searchengine.UpdateIndex | O(len(doc)) | O(len(doc)) | exact | O(len(doc)) | O(len(doc)) | exact |  | ya_algo sprint 4 final 1; author's claim "Построение поискового индекса - O(N)" over the words of the documents. Pinned against len(doc) because the word count is not a size bigo can name; words ≤ characters, so the pin is sound in the direction that matters. |
