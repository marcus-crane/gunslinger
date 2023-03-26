from django.urls import path
from rest_framework.urlpatterns import format_suffix_patterns
from liveplayer.views import MediaViewSet, api_root

media_list = MediaViewSet.as_view({
    'get': 'list'
})
media_detail = MediaViewSet.as_view({
    'get': 'retrieve'
})

urlpatterns = format_suffix_patterns([
    path('', api_root),
    path('history/', media_list, name='history'),
    path('history/<int:pk>/', media_detail)
])