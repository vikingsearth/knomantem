import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import '../models/user.dart';
import '../models/space.dart';
import '../models/page.dart';
import '../models/freshness.dart';
import '../models/search_result.dart';
import '../models/edge.dart';
import '../models/tag.dart';

const _baseUrl = 'http://localhost:8080/api/v1';
const _accessTokenKey = 'access_token';
const _refreshTokenKey = 'refresh_token';

class ApiException implements Exception {
  final String message;
  final int? statusCode;

  const ApiException(this.message, {this.statusCode});

  @override
  String toString() => 'ApiException($statusCode): $message';
}

class ApiService {
  final Dio _dio;
  final FlutterSecureStorage _storage;

  ApiService({Dio? dio, FlutterSecureStorage? storage})
      : _dio = dio ?? Dio(BaseOptions(baseUrl: _baseUrl)),
        _storage = storage ?? const FlutterSecureStorage() {
    _dio.interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) async {
        final token = await _storage.read(key: _accessTokenKey);
        if (token != null) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        handler.next(options);
      },
      onError: (error, handler) async {
        if (error.response?.statusCode == 401) {
          try {
            await _refreshToken();
            final token = await _storage.read(key: _accessTokenKey);
            if (token != null) {
              error.requestOptions.headers['Authorization'] = 'Bearer $token';
              final response = await _dio.fetch(error.requestOptions);
              handler.resolve(response);
              return;
            }
          } catch (_) {}
        }
        handler.next(error);
      },
    ));
  }

  Future<void> _refreshToken() async {
    final refreshToken = await _storage.read(key: _refreshTokenKey);
    if (refreshToken == null) throw const ApiException('No refresh token');

    final response = await _dio.post('/auth/refresh', data: {
      'refresh_token': refreshToken,
    });
    final data = response.data['data'] as Map<String, dynamic>;
    await _storage.write(
        key: _accessTokenKey, value: data['access_token'] as String);
    await _storage.write(
        key: _refreshTokenKey, value: data['refresh_token'] as String);
  }

  Future<T> _handleRequest<T>(Future<Response> Function() request,
      T Function(dynamic data) parser) async {
    try {
      final response = await request();
      return parser(response.data);
    } on DioException catch (e) {
      final msg = e.response?.data?['error']?['message'] as String? ??
          e.message ??
          'Network error';
      throw ApiException(msg, statusCode: e.response?.statusCode);
    }
  }

  // ─── Auth ───────────────────────────────────────────────────────────────────

  Future<({User user, String accessToken, String refreshToken})> login(
      String email, String password) async {
    return _handleRequest(
      () => _dio.post('/auth/login', data: {
        'email': email,
        'password': password,
      }),
      (data) {
        final d = (data as Map<String, dynamic>)['data'] as Map<String, dynamic>;
        return (
          user: User.fromJson(d['user'] as Map<String, dynamic>),
          accessToken: d['access_token'] as String,
          refreshToken: d['refresh_token'] as String,
        );
      },
    );
  }

  Future<({User user, String accessToken, String refreshToken})> register(
      String email, String displayName, String password) async {
    return _handleRequest(
      () => _dio.post('/auth/register', data: {
        'email': email,
        'display_name': displayName,
        'password': password,
      }),
      (data) {
        final d = (data as Map<String, dynamic>)['data'] as Map<String, dynamic>;
        return (
          user: User.fromJson(d['user'] as Map<String, dynamic>),
          accessToken: d['access_token'] as String,
          refreshToken: d['refresh_token'] as String,
        );
      },
    );
  }

  Future<User> getMe() async {
    return _handleRequest(
      () => _dio.get('/auth/me'),
      (data) => User.fromJson(
          ((data as Map<String, dynamic>)['data']) as Map<String, dynamic>),
    );
  }

  Future<void> saveTokens(String accessToken, String refreshToken) async {
    await _storage.write(key: _accessTokenKey, value: accessToken);
    await _storage.write(key: _refreshTokenKey, value: refreshToken);
  }

  Future<void> clearTokens() async {
    await _storage.delete(key: _accessTokenKey);
    await _storage.delete(key: _refreshTokenKey);
  }

  Future<String?> getAccessToken() => _storage.read(key: _accessTokenKey);

  // ─── Spaces ──────────────────────────────────────────────────────────────────

  Future<List<Space>> getSpaces({int limit = 50}) async {
    return _handleRequest(
      () => _dio.get('/spaces', queryParameters: {'limit': limit}),
      (data) {
        final list =
            ((data as Map<String, dynamic>)['data'] as List? ?? []);
        return list
            .map((s) => Space.fromJson(s as Map<String, dynamic>))
            .toList();
      },
    );
  }

  Future<Space> getSpace(String id) async {
    return _handleRequest(
      () => _dio.get('/spaces/$id'),
      (data) => Space.fromJson(
          ((data as Map<String, dynamic>)['data']) as Map<String, dynamic>),
    );
  }

  Future<Space> createSpace({
    required String name,
    String? description,
    String? icon,
  }) async {
    return _handleRequest(
      () => _dio.post('/spaces', data: {
        'name': name,
        if (description != null) 'description': description,
        if (icon != null) 'icon': icon,
        'settings': {},
      }),
      (data) => Space.fromJson(
          ((data as Map<String, dynamic>)['data']) as Map<String, dynamic>),
    );
  }

  Future<Space> updateSpace(String id,
      {String? name, String? description, String? icon}) async {
    return _handleRequest(
      () => _dio.put('/spaces/$id', data: {
        if (name != null) 'name': name,
        if (description != null) 'description': description,
        if (icon != null) 'icon': icon,
      }),
      (data) => Space.fromJson(
          ((data as Map<String, dynamic>)['data']) as Map<String, dynamic>),
    );
  }

  Future<void> deleteSpace(String id) async {
    await _handleRequest(
      () => _dio.delete('/spaces/$id'),
      (_) => null,
    );
  }

  // ─── Pages ───────────────────────────────────────────────────────────────────

  Future<List<PageSummary>> getPages(String spaceId) async {
    return _handleRequest(
      () => _dio.get('/spaces/$spaceId/pages',
          queryParameters: {'format': 'flat'}),
      (data) {
        final list =
            ((data as Map<String, dynamic>)['data'] as List? ?? []);
        return list
            .map((p) => PageSummary.fromJson(p as Map<String, dynamic>))
            .toList();
      },
    );
  }

  Future<PageDetail> getPage(String id) async {
    return _handleRequest(
      () => _dio.get('/pages/$id'),
      (data) => PageDetail.fromJson(
          ((data as Map<String, dynamic>)['data']) as Map<String, dynamic>),
    );
  }

  Future<PageDetail> createPage({
    required String spaceId,
    required String title,
    String? parentId,
    Map<String, dynamic>? content,
    String? icon,
    int position = 0,
  }) async {
    return _handleRequest(
      () => _dio.post('/spaces/$spaceId/pages', data: {
        'title': title,
        if (parentId != null) 'parent_id': parentId,
        'content': content ??
            {
              'type': 'doc',
              'content': [
                {
                  'type': 'heading',
                  'attrs': {'level': 1},
                  'content': [
                    {'type': 'text', 'text': title}
                  ],
                }
              ]
            },
        if (icon != null) 'icon': icon,
        'position': position,
        'is_template': false,
      }),
      (data) => PageDetail.fromJson(
          ((data as Map<String, dynamic>)['data']) as Map<String, dynamic>),
    );
  }

  Future<PageDetail> updatePage(String id,
      {String? title,
      Map<String, dynamic>? content,
      String? icon,
      String? changeSummary}) async {
    return _handleRequest(
      () => _dio.put('/pages/$id', data: {
        if (title != null) 'title': title,
        if (content != null) 'content': content,
        if (icon != null) 'icon': icon,
        if (changeSummary != null) 'change_summary': changeSummary,
      }),
      (data) => PageDetail.fromJson(
          ((data as Map<String, dynamic>)['data']) as Map<String, dynamic>),
    );
  }

  Future<void> deletePage(String id) async {
    await _handleRequest(
      () => _dio.delete('/pages/$id'),
      (_) => null,
    );
  }

  Future<void> movePage(String id,
      {required String? parentId, required int position}) async {
    await _handleRequest(
      () => _dio.put('/pages/$id/move', data: {
        'parent_id': parentId,
        'position': position,
      }),
      (_) => null,
    );
  }

  // ─── Search ──────────────────────────────────────────────────────────────────

  Future<SearchResponse> search({
    required String q,
    String? spaceId,
    List<String>? tags,
    String? freshness,
    String? from,
    String? to,
    String sort = 'relevance',
    String? cursor,
    int limit = 20,
  }) async {
    return _handleRequest(
      () => _dio.get('/search', queryParameters: {
        'q': q,
        if (spaceId != null) 'space': spaceId,
        if (tags != null && tags.isNotEmpty) 'tags': tags.join(','),
        if (freshness != null) 'freshness': freshness,
        if (from != null) 'from': from,
        if (to != null) 'to': to,
        'sort': sort,
        if (cursor != null) 'cursor': cursor,
        'limit': limit,
      }),
      (data) => SearchResponse.fromJson(data as Map<String, dynamic>),
    );
  }

  // ─── Freshness ───────────────────────────────────────────────────────────────

  Future<FreshnessInfo> getPageFreshness(String pageId) async {
    return _handleRequest(
      () => _dio.get('/pages/$pageId/freshness'),
      (data) => FreshnessInfo.fromJson(
          ((data as Map<String, dynamic>)['data']) as Map<String, dynamic>),
    );
  }

  Future<FreshnessInfo> verifyPage(String pageId, {String? notes}) async {
    return _handleRequest(
      () => _dio.post('/pages/$pageId/freshness/verify', data: {
        if (notes != null) 'notes': notes,
      }),
      (data) => FreshnessInfo.fromJson(
          ((data as Map<String, dynamic>)['data']) as Map<String, dynamic>),
    );
  }

  Future<({FreshnessSummary summary, List<FreshnessDashboardItem> pages})>
      getFreshnessDashboard({
    String? status,
    String sort = 'score',
    int limit = 20,
  }) async {
    return _handleRequest(
      () => _dio.get('/freshness/dashboard', queryParameters: {
        if (status != null) 'status': status,
        'sort': sort,
        'limit': limit,
      }),
      (data) {
        final d =
            ((data as Map<String, dynamic>)['data']) as Map<String, dynamic>;
        final summary =
            FreshnessSummary.fromJson(d['summary'] as Map<String, dynamic>);
        final pages = (d['pages'] as List? ?? [])
            .map((p) =>
                FreshnessDashboardItem.fromJson(p as Map<String, dynamic>))
            .toList();
        return (summary: summary, pages: pages);
      },
    );
  }

  // ─── Graph ───────────────────────────────────────────────────────────────────

  Future<GraphData> exploreGraph({
    required String rootId,
    int depth = 2,
    String? edgeType,
    int limit = 100,
  }) async {
    return _handleRequest(
      () => _dio.get('/graph/explore', queryParameters: {
        'root': rootId,
        'depth': depth,
        if (edgeType != null) 'edge_type': edgeType,
        'limit': limit,
      }),
      (data) => GraphData.fromJson(data as Map<String, dynamic>),
    );
  }

  Future<Map<String, dynamic>> getPageGraph(String pageId) async {
    return _handleRequest(
      () => _dio.get('/pages/$pageId/graph'),
      (data) => (data as Map<String, dynamic>)['data'] as Map<String, dynamic>,
    );
  }

  Future<GraphEdge> createEdge(String sourcePageId,
      {required String targetPageId, required String edgeType}) async {
    return _handleRequest(
      () => _dio.post('/pages/$sourcePageId/graph/edges', data: {
        'target_page_id': targetPageId,
        'edge_type': edgeType,
        'metadata': {},
      }),
      (data) => GraphEdge.fromJson(
          ((data as Map<String, dynamic>)['data']) as Map<String, dynamic>),
    );
  }

  // ─── Tags ────────────────────────────────────────────────────────────────────

  Future<List<Tag>> getTags({String? q, int limit = 50}) async {
    return _handleRequest(
      () => _dio.get('/tags', queryParameters: {
        if (q != null) 'q': q,
        'limit': limit,
      }),
      (data) {
        final list = ((data as Map<String, dynamic>)['data'] as List? ?? []);
        return list.map((t) => Tag.fromJson(t as Map<String, dynamic>)).toList();
      },
    );
  }

  Future<Tag> createTag(String name, {String? color}) async {
    return _handleRequest(
      () => _dio.post('/tags', data: {
        'name': name,
        if (color != null) 'color': color,
      }),
      (data) => Tag.fromJson(
          ((data as Map<String, dynamic>)['data']) as Map<String, dynamic>),
    );
  }

  Future<void> addTagsToPage(String pageId,
      List<({String tagId, double confidence})> tags) async {
    await _handleRequest(
      () => _dio.post('/pages/$pageId/tags', data: {
        'tags': tags
            .map((t) => {
                  'tag_id': t.tagId,
                  'confidence_score': t.confidence,
                })
            .toList(),
      }),
      (_) => null,
    );
  }
}

// ─── Providers ───────────────────────────────────────────────────────────────

final apiServiceProvider = Provider<ApiService>((ref) => ApiService());
