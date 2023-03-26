from liveplayer.models import MediaItem
from liveplayer.serializers import MediaItemSerializer
from rest_framework import generics, viewsets
from rest_framework.decorators import api_view
from rest_framework.response import Response
from rest_framework.reverse import reverse

class MediaViewSet(viewsets.ReadOnlyModelViewSet):
    queryset = MediaItem.objects.all()
    serializer_class = MediaItemSerializer

@api_view(['GET'])
def api_root(request, format=None):
    return Response({
        'history': reverse('history', request=request, format=format),
    })