# bigo canonical algorithm corpus — oracle golden

GENERATED — do not edit; regenerate with `task corpus`.

Literature-pinned worst-case bounds vs unaided inference. `exact` = inference
matches the literature; `loose` = sound but imprecise (a graduation target);
`top` = unverifiable (the annotate-or-trust evidence rows). A `wrong` never
appears here: it fails the build. Algorithms considered and kept out are in
[EXCLUSIONS.md](EXCLUSIONS.md). **This is not a coverage metric** — read
composition, not a percentage.

**Entries: 40**

## Time statuses

| Status | Count |
|---|---|
| exact | 14 |
| loose | 0 |
| top | 26 |

## Space statuses (pinned entries only)

| Status | Count |
|---|---|
| exact | 20 |
| loose | 1 |
| top | 19 |

## Per family

| Family | Entries |
|---|---|
| dandc | 6 |
| graphs | 6 |
| matrix | 5 |
| searching | 5 |
| sorting | 10 |
| stringops | 8 |

## Entries

| Function | Time pin | Time got | Status | Space pin | Space got | Status | Cause | Source |
|---|---|---|---|---|---|---|---|---|
| dandc.CountInversions | O(len(s) log(len(s))) | unverifiable | top | O(len(s)) | unverifiable | top | call | CLRS problem 2-4; en.wikipedia.org/wiki/Counting_inversions |
| dandc.MajorityDC | O(len(s) log(len(s))) | unverifiable | top | O(log(len(s))) | unverifiable | top | call | CLRS-style D&C; en.wikipedia.org/wiki/Boyer%E2%80%93Moore_majority_vote_algorithm (D&C alternative, bound reference) |
| dandc.MaxMinDC | O(len(s)) | unverifiable | top | O(log(len(s))) | unverifiable | top | call | www.geeksforgeeks.org/maximum-and-minimum-in-an-array/ (tournament method, bound reference) |
| dandc.MaxSubarrayDC | O(len(s) log(len(s))) | unverifiable | top | O(log(len(s))) | unverifiable | top | call | CLRS §4.1; en.wikipedia.org/wiki/Maximum_subarray_problem |
| dandc.PeakElement | O(log(len(s))) | O(log(len(s))) | exact | O(1) | O(1) | exact |  | jeffe.cs.illinois.edu/teaching/algorithms/ (recursion notes); www.geeksforgeeks.org/find-a-peak-in-a-given-array/ (bound reference) |
| dandc.PowerDC | O(log(b)) | O(log(b)) | exact | O(log(b)) | O(log(b)) | exact |  | CLRS §31.6 (repeated squaring); en.wikipedia.org/wiki/Exponentiation_by_squaring |
| graphs.BFS | O(len(adj)^2) | unverifiable | top | O(len(adj)) | unverifiable | top | loop | CLRS §22.2 — O(V+E), pinned at the E≤n² worst case |
| graphs.Components | O(len(adj)^2) | unverifiable | top | O(len(adj)) | unverifiable | top | loop | CLRS §21 intro — O(V+E) via repeated DFS, pinned at the E≤n² worst case |
| graphs.DFSIter | O(len(adj)^2) | unverifiable | top | O(len(adj)) | unverifiable | top | loop | CLRS §22.3 — O(V+E), pinned at the E≤n² worst case |
| graphs.DFSRec | O(len(adj)^2) | unverifiable | top | O(len(adj)) | unverifiable | top | loop | CLRS §22.3 (recursive form) — stack depth ≤ n |
| graphs.FloydWarshall | O(len(dist)^3) | O(len(dist)^3) | exact | O(1) | O(1) | exact |  | CLRS §25.2; en.wikipedia.org/wiki/Floyd%E2%80%93Warshall_algorithm |
| graphs.TopoSortKahn | O(len(adj)^2) | unverifiable | top | O(len(adj)) | unverifiable | top | loop | CLRS §22.4 / Kahn 1962 — O(V+E), pinned at the E≤n² worst case |
| matrix.Mul | O(len(a)^3) | O(len(a)^3) | exact | O(len(a)^2) | unverifiable | top |  | CLRS §4.2 (naive); en.wikipedia.org/wiki/Matrix_multiplication_algorithm |
| matrix.Rotate90 | O(len(m)^2) | O(len(m)^2) | exact | O(1) | O(1) | exact |  | www.geeksforgeeks.org/rotate-a-matrix-by-90-degree-in-clockwise-direction/ (bound reference) |
| matrix.SearchSorted | O(len(m)) | unverifiable | top | O(1) | O(1) | exact | loop | www.geeksforgeeks.org/search-in-row-wise-and-column-wise-sorted-matrix/ (bound reference) |
| matrix.SpiralOrder | O(len(m)^2) | unverifiable | top | O(len(m)^2) | unverifiable | top | loop | www.geeksforgeeks.org/print-a-given-matrix-in-spiral-form/ (bound reference) |
| matrix.TransposeInPlace | O(len(m)^2) | O(len(m)^2) | exact | O(1) | O(1) | exact |  | en.wikipedia.org/wiki/Transpose |
| searching.BinarySearch | O(log(len(s))) | unverifiable | top | O(1) | O(1) | exact | loop | CLRS ex. 2.3-5; en.wikipedia.org/wiki/Binary_search_algorithm |
| searching.BinarySearchRec | O(log(len(s))) | O(log(len(s))) | exact | O(log(len(s))) | O(log(len(s))) | exact |  | CLRS ex. 2.3-5 (recursive form) |
| searching.FirstOccurrence | O(log(len(s))) | unverifiable | top | O(1) | O(1) | exact | loop | en.wikipedia.org/wiki/Binary_search_algorithm (leftmost variant) |
| searching.LinearSearch | O(len(s)) | O(len(s)) | exact | O(1) | O(1) | exact |  | en.wikipedia.org/wiki/Linear_search |
| searching.SearchRotated | O(log(len(s))) | unverifiable | top | O(1) | O(1) | exact | loop | www.geeksforgeeks.org/search-an-element-in-a-sorted-and-pivoted-array/ (bound reference) |
| sorting.BubbleSort | O(len(s)^2) | O(len(s)^2) | exact | O(1) | O(1) | exact |  | CLRS problem 2-2; en.wikipedia.org/wiki/Bubble_sort |
| sorting.BucketSort | O(len(s)^2) | unverifiable | top | O(len(s)) | unverifiable | top | loop | CLRS §8.4 (worst case); en.wikipedia.org/wiki/Bucket_sort |
| sorting.CountingSort | O(k + len(s)) | unverifiable | top | O(k + len(s)) | unverifiable | top | loop | CLRS §8.2; en.wikipedia.org/wiki/Counting_sort |
| sorting.HeapSort | O(len(s) log(len(s))) | unverifiable | top | O(1) | O(len(s)) | loose | call | CLRS §6.4; en.wikipedia.org/wiki/Heapsort |
| sorting.InsertionSort | O(len(s)^2) | O(len(s)^2) | exact | O(1) | O(1) | exact |  | CLRS §2.1; en.wikipedia.org/wiki/Insertion_sort (worst case) |
| sorting.MergeSort | O(len(s) log(len(s))) | unverifiable | top | O(len(s)) | unverifiable | top | call | CLRS §2.3.1; en.wikipedia.org/wiki/Merge_sort |
| sorting.QuickSort | O(len(s)^2) | unverifiable | top | O(len(s)) | unverifiable | top | call | CLRS §7; en.wikipedia.org/wiki/Quicksort (worst case) |
| sorting.RadixSortLSD | O(len(s)) | unverifiable | top | O(len(s)) | unverifiable | top | loop | CLRS §8.3 (fixed d, k); en.wikipedia.org/wiki/Radix_sort |
| sorting.SelectionSort | O(len(s)^2) | O(len(s)^2) | exact | O(1) | O(1) | exact |  | CLRS ex. 2.2-2; en.wikipedia.org/wiki/Selection_sort |
| sorting.ShellSort | O(len(s)^2) | unverifiable | top | O(1) | O(1) | exact | loop | en.wikipedia.org/wiki/Shellsort (Shell's sequence, worst case) |
| stringops.AreAnagrams | O(len(a) + len(b)) | unverifiable | top | O(1) | O(1) | exact | loop | www.geeksforgeeks.org/check-whether-two-strings-are-anagram-of-each-other/ (bound reference) |
| stringops.CommonPrefix | O(len(a)) | O(len(a)) | exact | O(1) | O(1) | exact |  | en.wikipedia.org/wiki/LCP_array (pairwise base case, bound reference) |
| stringops.IsPalindrome | O(len(s)) | O(len(s)) | exact | O(1) | O(1) | exact |  | ru.algorithmica.org (strings, bound reference); en.wikipedia.org/wiki/Palindrome |
| stringops.KMPSearch | O(len(pat) + len(text)) | unverifiable | top | O(len(pat)) | unverifiable | top | loop | CLRS §32.4; en.wikipedia.org/wiki/Knuth%E2%80%93Morris%E2%80%93Pratt_algorithm |
| stringops.NaiveSearch | O(len(pat) len(text)) | unverifiable | top | O(1) | O(1) | exact | loop | CLRS §32.1; en.wikipedia.org/wiki/String-searching_algorithm |
| stringops.RabinKarp | O(len(pat) len(text)) | unverifiable | top | O(1) | O(1) | exact | loop | CLRS §32.2 (worst case); en.wikipedia.org/wiki/Rabin%E2%80%93Karp_algorithm |
| stringops.Reverse | O(len(s)) | O(len(s)) | exact | O(len(s)) | unverifiable | top |  | ru.wikibooks.org/wiki/Реализации_алгоритмов (strings, bound reference) |
| stringops.RunLengthEncode | O(len(s)) | unverifiable | top | O(len(s)) | unverifiable | top | loop | en.wikipedia.org/wiki/Run-length_encoding |
